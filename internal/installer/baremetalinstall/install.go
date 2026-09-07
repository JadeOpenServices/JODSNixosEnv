// Package baremetalinstall performs the final NixOS installation into a
// previously prepared fresh target mounted at /mnt.
//
// Disk selection, partitioning, resizing, encryption, filesystem creation, and
// mounting belong to earlier installer stages. This package deliberately does
// none of them. In particular, it preserves the post-GJAL-31 rule that normal
// installation never assumes it owns or may recreate the whole target disk.
package baremetalinstall

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const targetRoot = "/mnt"

var hostnamePattern = regexp.MustCompile(
	`^[A-Za-z0-9][A-Za-z0-9.-]{0,252}$`,
)

type Input struct {
	Repo     string
	Hostname string
	Out      io.Writer
}

type Result struct {
	Target string
	Stage  string
}

type runner interface {
	Run(context.Context, io.Reader, io.Writer, io.Writer, string, ...string) error
	Output(context.Context, string, ...string) ([]byte, error)
}

type execRunner struct{}

func (execRunner) Run(
	ctx context.Context,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	name string,
	args ...string,
) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = stdin
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

func Install(ctx context.Context, input Input) (Result, error) {
	return install(ctx, input, execRunner{})
}

func install(
	ctx context.Context,
	input Input,
	r runner,
) (Result, error) {
	if input.Out == nil {
		return Result{}, fmt.Errorf("installation output writer is required")
	}

	repo, target, err := validateSource(input.Repo, input.Hostname)
	if err != nil {
		return Result{}, err
	}

	fmt.Fprintln(input.Out, "STAGE: validating-target")

	if err := verifyExactMount(ctx, r, targetRoot); err != nil {
		return Result{}, fmt.Errorf("verify target root: %w", err)
	}
	if err := verifyExactMount(ctx, r, filepath.Join(targetRoot, "boot")); err != nil {
		return Result{}, fmt.Errorf("verify target boot mount: %w", err)
	}

	configBefore, err := snapshotInstallerConfig(repo)
	if err != nil {
		return Result{}, err
	}

	fmt.Fprintf(
		input.Out,
		"STAGE: installing target=%s root=%s\n",
		target,
		targetRoot,
	)

	// --no-root-passwd ensures nixos-install never opens a password prompt or
	// tries to move a secret through this stage. GjallarOS owns its configured
	// credential policy separately.
	//
	// No JODS service, endpoint, enrollment command, or installation-complete
	// marker is touched here. Management activation happens only in its later
	// explicitly gated lifecycle.
	if err := r.Run(
		ctx,
		nil,
		input.Out,
		input.Out,
		"sudo",
		"nixos-install",
		"--root",
		targetRoot,
		"--flake",
		target,
		"--no-root-passwd",
	); err != nil {
		return Result{}, fmt.Errorf("nixos-install failed: %w", err)
	}

	if err := verifyInstallerConfig(repo, configBefore); err != nil {
		return Result{}, err
	}

	// Successful nixos-install must leave the target system profile behind.
	systemProfile := filepath.Join(
		targetRoot,
		"nix",
		"var",
		"nix",
		"profiles",
		"system",
	)
	info, err := os.Stat(systemProfile)
	if err != nil {
		return Result{}, fmt.Errorf(
			"verify installed system profile %s: %w",
			systemProfile,
			err,
		)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return Result{}, fmt.Errorf(
			"installed system profile is not a symlink: %s",
			systemProfile,
		)
	}

	fmt.Fprintln(input.Out, "STAGE: os-installed")
	fmt.Fprintf(
		input.Out,
		"PASS: GjallarOS installed into %s using %s\n",
		targetRoot,
		target,
	)

	return Result{
		Target: target,
		Stage:  "os-installed",
	}, nil
}

func validateSource(repo, hostname string) (string, string, error) {
	if strings.TrimSpace(repo) == "" {
		return "", "", fmt.Errorf("repository path is required")
	}

	root, err := filepath.Abs(repo)
	if err != nil {
		return "", "", fmt.Errorf("resolve repository: %w", err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", "", fmt.Errorf("resolve repository: %w", err)
	}

	info, err := os.Stat(filepath.Join(root, "flake.nix"))
	if err != nil {
		return "", "", fmt.Errorf("flake.nix not found: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", "", fmt.Errorf("flake.nix is not a regular file")
	}

	if !hostnamePattern.MatchString(hostname) ||
		hostname == "." ||
		hostname == ".." {
		return "", "", fmt.Errorf(
			"invalid configuration hostname: %q",
			hostname,
		)
	}

	return root, root + "#" + hostname, nil
}

func verifyExactMount(
	ctx context.Context,
	r runner,
	mountPoint string,
) error {
	raw, err := r.Output(
		ctx,
		"findmnt",
		"-nro",
		"TARGET",
		"--mountpoint",
		mountPoint,
	)
	if err != nil {
		return fmt.Errorf("%s is not mounted: %w", mountPoint, err)
	}

	actual := filepath.Clean(strings.TrimSpace(string(raw)))
	if actual != mountPoint {
		return fmt.Errorf(
			"mount target mismatch: expected %s, got %q",
			mountPoint,
			strings.TrimSpace(string(raw)),
		)
	}

	return nil
}

type configSnapshot struct {
	Path   string
	Exists bool
	Sum    [sha256.Size]byte
}

func snapshotInstallerConfig(repo string) ([]configSnapshot, error) {
	paths := []string{
		filepath.Join(repo, "settings.nix"),
		filepath.Join(repo, "user.config.json"),
	}

	result := make([]configSnapshot, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			result = append(result, configSnapshot{
				Path:   path,
				Exists: false,
			})
			continue
		}
		if err != nil {
			return nil, fmt.Errorf(
				"snapshot installer config %s: %w",
				path,
				err,
			)
		}

		result = append(result, configSnapshot{
			Path:   path,
			Exists: true,
			Sum:    sha256.Sum256(data),
		})
	}

	return result, nil
}

func verifyInstallerConfig(
	repo string,
	before []configSnapshot,
) error {
	expected := map[string]configSnapshot{}
	for _, snapshot := range before {
		expected[snapshot.Path] = snapshot
	}

	for _, name := range []string{"settings.nix", "user.config.json"} {
		path := filepath.Join(repo, name)
		prior := expected[path]

		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			if prior.Exists {
				return fmt.Errorf(
					"installer config disappeared during installation: %s",
					path,
				)
			}
			continue
		}
		if err != nil {
			return fmt.Errorf(
				"verify installer config %s: %w",
				path,
				err,
			)
		}
		if !prior.Exists {
			return fmt.Errorf(
				"installer config unexpectedly appeared during installation: %s",
				path,
			)
		}

		after := sha256.Sum256(data)
		if !bytes.Equal(after[:], prior.Sum[:]) {
			return fmt.Errorf(
				"installer config changed during installation: %s",
				path,
			)
		}
	}

	return nil
}
