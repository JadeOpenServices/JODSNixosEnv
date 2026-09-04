package secrets

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

type Status struct {
	SecretsExist, KeyExists, SOPSAvailable, Decrypts bool
	KeyPath                                          string
}

func EnsureAgeKey(ctx context.Context, username string) (keyPath, recipient string, err error) {
	account, err := user.Lookup(username)
	if err != nil {
		return "", "", err
	}
	keyPath = filepath.Join(account.HomeDir, ".config", "sops", "age", "keys.txt")
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(keyPath), 0700); err != nil {
			return "", "", err
		}
		tmp, err := os.CreateTemp(filepath.Dir(keyPath), ".age-key-*")
		if err != nil {
			return "", "", err
		}
		tmpPath := tmp.Name()
		tmp.Close()
		os.Remove(tmpPath)
		defer os.Remove(tmpPath)
		cmd := exec.CommandContext(ctx, "age-keygen", "-o", tmpPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return "", "", fmt.Errorf("generate age key: %w", err)
		}
		if err := os.Chmod(tmpPath, 0600); err != nil {
			return "", "", err
		}
		if err := os.Rename(tmpPath, keyPath); err != nil {
			return "", "", err
		}
	}
	f, err := os.Open(keyPath)
	if err != nil {
		return "", "", err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "public key:") {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				recipient = fields[len(fields)-1]
				break
			}
		}
	}
	if !strings.HasPrefix(recipient, "age1") {
		return "", "", fmt.Errorf("could not read public recipient from %s", keyPath)
	}
	return keyPath, recipient, nil
}

func EncryptScrobbling(ctx context.Context, repo, recipient, lastfm, listenbrainz string) error {
	if lastfm == "" && listenbrainz == "" {
		return fmt.Errorf("no scrobbling credentials supplied")
	}
	if err := os.MkdirAll(filepath.Join(repo, "secrets"), 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp("", "gjallar-scrobbling-*.yaml")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	_ = tmp.Chmod(0600)
	escape := func(v string) string { return strings.ReplaceAll(v, "'", "''") }
	if lastfm != "" {
		fmt.Fprintf(tmp, "lastfm: '%s'\n", escape(lastfm))
	}
	if listenbrainz != "" {
		fmt.Fprintf(tmp, "listenbrainz: '%s'\n", escape(listenbrainz))
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	target := filepath.Join(repo, "secrets", "default.yaml")
	encrypted, err := os.CreateTemp(filepath.Dir(target), ".default-encrypted-*.yaml")
	if err != nil {
		return err
	}
	encryptedPath := encrypted.Name()
	encrypted.Close()
	defer os.Remove(encryptedPath)
	cmd := exec.CommandContext(ctx, "sops", "--encrypt", "--age", recipient, "--output", encryptedPath, name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("encrypt scrobbling credentials: %w", err)
	}
	if err := os.Rename(encryptedPath, target); err != nil {
		return fmt.Errorf("replace encrypted credentials: %w", err)
	}
	config := []byte("# Managed by gjallarctl.\ncreation_rules:\n  - path_regex: secrets/[^/]+\\.(yaml|json|env|ini)$\n    age: " + recipient + "\n")
	return atomicWrite(filepath.Join(repo, ".sops.yaml"), config, 0644)
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".sops-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

func Check(ctx context.Context, repo, username string) (Status, error) {
	account, err := user.Lookup(username)
	if err != nil {
		return Status{}, fmt.Errorf("look up user %q: %w", username, err)
	}
	key := filepath.Join(account.HomeDir, ".config", "sops", "age", "keys.txt")
	secret := filepath.Join(repo, "secrets", "default.yaml")
	status := Status{KeyPath: key}
	if info, err := os.Stat(secret); err == nil && !info.IsDir() {
		status.SecretsExist = true
	} else if err != nil && !os.IsNotExist(err) {
		return status, err
	}
	if info, err := os.Stat(key); err == nil && !info.IsDir() {
		status.KeyExists = true
	}
	if _, err := exec.LookPath("sops"); err == nil {
		status.SOPSAvailable = true
	}
	if status.SecretsExist && status.KeyExists && status.SOPSAvailable {
		cmd := exec.CommandContext(ctx, "sops", "--decrypt", secret)
		cmd.Stdout = nil
		cmd.Stderr = nil
		status.Decrypts = cmd.Run() == nil
	}
	return status, nil
}
