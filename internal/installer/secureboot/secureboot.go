package secureboot

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
)

const ArchivePath = "/var/lib/gjallarOS/recovery/secure-boot-keys.tar.enc"
const EnrollmentMarkerPath = "/var/lib/gjallarOS/secure-boot-enrollment-armed"

type Recovery struct {
	ArchivePath string
	Passphrase  string
}

func Provision(ctx context.Context) (Recovery, error) {
	passphrase, err := generatePassphrase()
	if err != nil {
		return Recovery{}, err
	}
	passFile, err := os.CreateTemp("", "gjallar-secure-boot-passphrase-*")
	if err != nil {
		return Recovery{}, err
	}
	passPath := passFile.Name()
	defer os.Remove(passPath)
	if err := passFile.Chmod(0600); err != nil {
		passFile.Close()
		return Recovery{}, err
	}
	if _, err := passFile.WriteString(passphrase); err != nil {
		passFile.Close()
		return Recovery{}, err
	}
	if err := passFile.Close(); err != nil {
		return Recovery{}, err
	}

	if err := run(ctx, "sudo", "test", "-f", "/var/lib/sbctl/keys/db/db.key"); err != nil {
		if err := run(ctx, "sudo", "sbctl", "create-keys"); err != nil {
			return Recovery{}, fmt.Errorf("create Secure Boot keys: %w", err)
		}
	}
	if err := run(ctx, "sudo", "install", "-d", "-m", "0700", "/var/lib/gjallarOS/recovery"); err != nil {
		return Recovery{}, err
	}
	raw := "/var/lib/gjallarOS/recovery/.secure-boot-keys.tar"
	defer func() { _ = run(context.Background(), "sudo", "rm", "-f", "--", raw) }()
	if err := run(ctx, "sudo", "tar", "-C", "/var/lib/sbctl", "-cf", raw, "."); err != nil {
		return Recovery{}, fmt.Errorf("archive Secure Boot keys: %w", err)
	}
	if err := run(ctx, "sudo", "openssl", "enc", "-aes-256-cbc", "-pbkdf2", "-salt", "-in", raw, "-out", ArchivePath, "-pass", "file:"+passPath); err != nil {
		_ = run(ctx, "sudo", "rm", "-f", "--", ArchivePath)
		return Recovery{}, fmt.Errorf("encrypt Secure Boot recovery archive: %w", err)
	}
	if err := run(ctx, "sudo", "chmod", "0600", ArchivePath); err != nil {
		return Recovery{}, err
	}
	return Recovery{ArchivePath: ArchivePath, Passphrase: passphrase}, nil
}

// VerifyAndArmEnrollment only arms the boot-time enrollment service after the
// installed EFI artifacts have been signed successfully.
func VerifyAndArmEnrollment(ctx context.Context) error {
	if err := run(ctx, "sudo", "sbctl", "verify"); err != nil {
		return fmt.Errorf("verify signed Secure Boot artifacts: %w", err)
	}
	if err := run(ctx, "sudo", "install", "-d", "-m", "0700", "/var/lib/gjallarOS"); err != nil {
		return fmt.Errorf("create Secure Boot state directory: %w", err)
	}
	if err := run(ctx, "sudo", "install", "-m", "0600", "/dev/null", EnrollmentMarkerPath); err != nil {
		return fmt.Errorf("arm firmware enrollment: %w", err)
	}
	return nil
}

func generatePassphrase() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
