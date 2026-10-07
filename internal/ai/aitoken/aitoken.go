// Package aitoken holds the bearer token the system uses to log in to a
// central AI server. The token never enters the Nix store or user.config.json:
// the installer or `gjallarctl ai set-token` leaves it root-only at Pending,
// and ai-endpoint-token-seal encrypts it with systemd-creds (TPM2 when
// present) into Sealed and removes Pending.
package aitoken

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	Pending = "/var/lib/gjallarOS/ai/endpoint-token"
	Sealed  = "/var/lib/gjallarOS/ai/endpoint-token.cred"
	// Name is the systemd credential name the AI services load.
	Name = "ai-endpoint-token"
)

const (
	minLength = 32
	maxLength = 4096
)

var tokenPattern = regexp.MustCompile(`^[A-Za-z0-9._~+/=-]+$`)

// Validate refuses tokens too short or too odd to be a generated secret.
func Validate(token string) error {
	switch {
	case token == "":
		return errors.New("AI server token is required")
	case len(token) < minLength:
		return fmt.Errorf("AI server token must be at least %d characters; generate one with `openssl rand -hex 32`", minLength)
	case len(token) > maxLength:
		return fmt.Errorf("AI server token must be at most %d characters", maxLength)
	case !tokenPattern.MatchString(token):
		return errors.New("AI server token may only contain A-Z a-z 0-9 . _ ~ + / = -")
	}
	return nil
}

// Read returns the token in path, which must be valid.
func Read(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read AI server token: %w", err)
	}
	token := strings.TrimSpace(string(raw))
	if err := Validate(token); err != nil {
		return "", err
	}
	return token, nil
}

// Store writes the token root-only to target via sudo. The token goes over
// stdin, never argv or a temporary file.
func Store(ctx context.Context, token, target string) error {
	if err := Validate(token); err != nil {
		return err
	}
	for _, args := range storeCommands(target) {
		cmd := exec.CommandContext(ctx, "sudo", args...)
		if args[len(args)-2] == "/dev/stdin" {
			cmd.Stdin = strings.NewReader(token + "\n")
		}
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("store AI server token: %w", err)
		}
	}
	return nil
}

func storeCommands(target string) [][]string {
	return [][]string{
		{"install", "-d", "-m", "0700", "-o", "root", "-g", "root", "--", filepath.Dir(target)},
		{"install", "-m", "0600", "-o", "root", "-g", "root", "--", "/dev/stdin", target},
	}
}
