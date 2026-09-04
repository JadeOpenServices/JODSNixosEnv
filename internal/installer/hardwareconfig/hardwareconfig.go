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
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) != 3 || parts[0] != "profiles" || parts[2] != "hardware-configuration.nix" || parts[1] == "" {
		return "", fmt.Errorf("invalid hardware target: %s", target)
	}
	return abs, nil
}

func Generate(ctx context.Context, repo, target string, now time.Time) (string, error) {
	target, err := ValidateTarget(repo, target)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return "", fmt.Errorf("create profile directory: %w", err)
	}
	backup := ""
	if _, err := os.Stat(target); err == nil {
		backup = target + ".bak." + now.Format("20060102150405")
		if err := copyFile(target, backup); err != nil {
			return "", err
		}
	}
	tmp, err := os.CreateTemp(filepath.Dir(target), ".hardware-configuration.nix-*")
	if err != nil {
		return backup, fmt.Errorf("create temporary hardware configuration: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	cmd := exec.CommandContext(ctx, "nixos-generate-config", "--show-hardware-config")
	cmd.Stdout = tmp
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		tmp.Close()
		return backup, fmt.Errorf("nixos-generate-config: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return backup, fmt.Errorf("sync hardware configuration: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return backup, fmt.Errorf("close hardware configuration: %w", err)
	}
	if err := os.Rename(tmpName, target); err != nil {
		return backup, fmt.Errorf("replace hardware configuration: %w", err)
	}
	return backup, nil
}

func copyFile(source, target string) error {
	in, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open hardware backup source: %w", err)
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("create hardware backup: %w", err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return fmt.Errorf("write hardware backup: %w", err)
	}
	if err := out.Sync(); err != nil {
		out.Close()
		return fmt.Errorf("sync hardware backup: %w", err)
	}
	return out.Close()
}
