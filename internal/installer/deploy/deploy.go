package deploy

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
)

var hostnamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.-]{0,252}$`)

func Target(repo, hostname string) (string, error) {
	root, err := filepath.Abs(repo)
	if err != nil {
		return "", fmt.Errorf("resolve repository: %w", err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("resolve repository: %w", err)
	}
	if _, err := os.Stat(filepath.Join(root, "flake.nix")); err != nil {
		return "", fmt.Errorf("flake.nix not found: %w", err)
	}
	if !hostnamePattern.MatchString(hostname) || hostname == "." || hostname == ".." {
		return "", fmt.Errorf("invalid configuration hostname: %q", hostname)
	}
	return "path:" + root + "#" + hostname, nil
}

func Apply(ctx context.Context, target string) error {
	if err := run(ctx, "sudo", "nixos-rebuild", "dry-build", "--flake", target, "--show-trace"); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	if err := run(ctx, "sudo", "nixos-rebuild", "boot", "--flake", target); err != nil {
		return fmt.Errorf("boot generation failed: %w", err)
	}
	return nil
}

func run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
