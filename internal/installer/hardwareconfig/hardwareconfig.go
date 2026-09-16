package hardwareconfig

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	freshTargetRoot = "/mnt"
	relativeTarget  = "generated/hardware.nix"
)

func Target(repo string) string {
	return filepath.Join(repo, filepath.FromSlash(relativeTarget))
}

type commandRunner interface {
	Run(context.Context, io.Writer, io.Writer, string, ...string) error
	Output(context.Context, string, ...string) ([]byte, error)
}

type execRunner struct{}

func (execRunner) Run(
	ctx context.Context,
	stdout io.Writer,
	stderr io.Writer,
	name string,
	args ...string,
) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

func (execRunner) Output(
	ctx context.Context,
	name string,
	args ...string,
) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}

func ValidateTarget(repo, target string) (string, error) {
	root, err := filepath.Abs(repo)
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("hardware target is outside repository: %s", target)
	}
	if filepath.ToSlash(rel) != relativeTarget {
		return "", fmt.Errorf(
			"invalid hardware target %s: expected %s",
			target,
			relativeTarget,
		)
	}
	return abs, nil
}

// Generate preserves the existing behavior: generate hardware configuration
// for the currently running machine.
func Generate(
	ctx context.Context,
	repo,
	target string,
	now time.Time,
) (string, error) {
	return generate(
		ctx,
		repo,
		target,
		now,
		"",
		execRunner{},
	)
}

// GenerateTarget generates hardware configuration for the prepared fresh
// installation mounted at /mnt.
//
// It deliberately does not alter flake.nix or any unrelated source file.
// Generated hardware state has one canonical machine-local target and is
// backed up before atomic replacement.
func GenerateTarget(
	ctx context.Context,
	repo,
	target string,
	now time.Time,
) (string, error) {
	return generate(
		ctx,
		repo,
		target,
		now,
		freshTargetRoot,
		execRunner{},
	)
}

func generate(
	ctx context.Context,
	repo,
	target string,
	now time.Time,
	root string,
	runner commandRunner,
) (string, error) {
	target, err := ValidateTarget(repo, target)
	if err != nil {
		return "", err
	}

	if root != "" {
		cleanRoot := filepath.Clean(root)
		isolatedRoot := strings.HasPrefix(
			filepath.Base(cleanRoot),
			"gjallar-hardware-root-",
		)

		if cleanRoot != freshTargetRoot && !isolatedRoot {
			return "", fmt.Errorf(
				"unsupported hardware discovery root %s",
				root,
			)
		}

		if cleanRoot == freshTargetRoot {
			if err := verifyFreshTargetMount(ctx, runner, root); err != nil {
				return "", err
			}
		}
	}

	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return "", fmt.Errorf("create hardware state directory: %w", err)
	}

	backup := ""
	if _, err := os.Stat(target); err == nil {
		backup = target + ".bak." + now.Format("20060102150405")
		if err := copyFile(target, backup); err != nil {
			return "", err
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("inspect hardware configuration target: %w", err)
	}

	tmp, err := os.CreateTemp(
		filepath.Dir(target),
		".generated.nix-*",
	)
	if err != nil {
		return backup, fmt.Errorf(
			"create temporary hardware configuration: %w",
			err,
		)
	}

	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	args := []string{}
	if root != "" {
		args = append(args, "--root", root)
	}
	args = append(args, "--show-hardware-config")

	commandArgs := append(
		[]string{"nixos-generate-config"},
		args...,
	)

	if err := runner.Run(
		ctx,
		tmp,
		os.Stderr,
		"sudo",
		commandArgs...,
	); err != nil {
		_ = tmp.Close()
		return backup, fmt.Errorf("nixos-generate-config: %w", err)
	}

	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return backup, fmt.Errorf(
			"sync hardware configuration: %w",
			err,
		)
	}
	if err := tmp.Close(); err != nil {
		return backup, fmt.Errorf(
			"close hardware configuration: %w",
			err,
		)
	}

	info, err := os.Stat(tmpName)
	if err != nil {
		return backup, fmt.Errorf(
			"inspect generated hardware configuration: %w",
			err,
		)
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		return backup, fmt.Errorf(
			"nixos-generate-config produced an empty or invalid hardware configuration",
		)
	}

	data, err := os.ReadFile(tmpName)
	if err != nil {
		return backup, fmt.Errorf(
			"read generated hardware configuration: %w",
			err,
		)
	}
	if !bytesContainNixModule(data) {
		return backup, fmt.Errorf(
			"generated hardware configuration does not look like a NixOS module",
		)
	}

	if err := os.Rename(tmpName, target); err != nil {
		return backup, fmt.Errorf(
			"replace hardware configuration: %w",
			err,
		)
	}

	return backup, nil
}

func verifyFreshTargetMount(
	ctx context.Context,
	runner commandRunner,
	root string,
) error {
	raw, err := runner.Output(
		ctx,
		"findmnt",
		"-nro",
		"TARGET",
		"--mountpoint",
		root,
	)
	if err != nil {
		return fmt.Errorf(
			"fresh target %s is not mounted: %w",
			root,
			err,
		)
	}

	if filepath.Clean(strings.TrimSpace(string(raw))) != root {
		return fmt.Errorf(
			"fresh target mount verification failed: expected %s, got %q",
			root,
			strings.TrimSpace(string(raw)),
		)
	}

	return nil
}

func bytesContainNixModule(data []byte) bool {
	text := strings.TrimSpace(string(data))
	return strings.Contains(text, "{") &&
		strings.Contains(text, "config") &&
		strings.Contains(text, "lib") &&
		strings.Contains(text, "pkgs")
}

func copyFile(source, target string) error {
	in, err := os.Open(source)
	if err != nil {
		return fmt.Errorf(
			"open hardware backup source: %w",
			err,
		)
	}
	defer in.Close()

	out, err := os.OpenFile(
		target,
		os.O_CREATE|os.O_EXCL|os.O_WRONLY,
		0600,
	)
	if err != nil {
		return fmt.Errorf("create hardware backup: %w", err)
	}

	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return fmt.Errorf("write hardware backup: %w", err)
	}
	if err := out.Sync(); err != nil {
		_ = out.Close()
		return fmt.Errorf("sync hardware backup: %w", err)
	}
	return out.Close()
}
