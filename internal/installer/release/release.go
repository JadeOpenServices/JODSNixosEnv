package release

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Policy struct {
	Release string `json:"release"`
}

func Inspect(repo, osRelease string) (expected, actual string, err error) {
	data, err := os.ReadFile(filepath.Join(repo, "deployment", "release-policy.json"))
	if err != nil {
		return "", "", fmt.Errorf("read release policy: %w", err)
	}
	var policy Policy
	if err := json.Unmarshal(data, &policy); err != nil {
		return "", "", fmt.Errorf("parse release policy: %w", err)
	}
	f, err := os.Open(osRelease)
	if err != nil {
		return "", "", fmt.Errorf("read OS release: %w", err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		key, value, ok := strings.Cut(scanner.Text(), "=")
		if ok && key == "VERSION_ID" {
			actual = strings.Trim(value, `"`)
		}
	}
	if policy.Release == "" || actual == "" {
		return "", "", fmt.Errorf("could not determine the NixOS release")
	}
	return policy.Release, actual, scanner.Err()
}

func Align(ctx context.Context, expected, nixosConfig string) error {
	url := "https://channels.nixos.org/nixos-" + expected
	for _, action := range [][]string{{"nix-channel", "--add", url, "nixos"}, {"nix-channel", "--update", "nixos"}, {"nixos-rebuild", "boot", "--upgrade", "-I", "nixos-config=" + nixosConfig}} {
		args := append([]string{action[0]}, action[1:]...)
		cmd := exec.CommandContext(ctx, "sudo", args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("sudo %s: %w", strings.Join(args, " "), err)
		}
	}
	return nil
}
