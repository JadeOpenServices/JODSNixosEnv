package secureboot

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const ArchivePath = "/var/lib/gjallarOS/recovery/secure-boot-keys.tar.enc"
const EnrollmentMarkerPath = "/var/lib/gjallarOS/secure-boot-enrollment-armed"
const FinalMarkerPath = "/var/lib/gjallarOS/secure-boot-enable-required"

type Continuation string

const (
	ContinuationNone   Continuation = "none"
	ContinuationEnroll Continuation = "enroll"
	ContinuationEnable Continuation = "enable"
)

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

// VerifyAndArmEnrollment validates the installed boot chain and determines the
// next Secure Boot transition.
//
// Lanzaboote intentionally keeps the external /boot/EFI/nixos/kernel-*.efi
// byte-identical to the Nix-store kernel. The signed generation stub records
// and verifies that kernel hash. Individually Authenticode-signing that loose
// kernel changes its bytes and causes Lanzaboote to abort with
// SECURITY_VIOLATION / "Kernel hash does not match".
func VerifyAndArmEnrollment(ctx context.Context) (Continuation, error) {
	if err := verifyBootArtifacts(ctx); err != nil {
		return ContinuationNone, err
	}

	inspection, err := Inspect(ctx)
	if err != nil {
		return ContinuationNone, fmt.Errorf(
			"inspect Secure Boot ownership before enrollment: %w", err,
		)
	}

	if err := run(
		ctx,
		"sudo", "install", "-d", "-m", "0700",
		"/var/lib/gjallarOS",
	); err != nil {
		return ContinuationNone, fmt.Errorf(
			"create Secure Boot state directory: %w", err,
		)
	}

	switch inspection.State {
	case StateGjallarManaged:
		if err := RecordOwnership(ctx, "enrolled"); err != nil {
			return ContinuationNone, err
		}

		if err := run(
			ctx,
			"sudo", "rm", "-f", "--",
			EnrollmentMarkerPath,
		); err != nil {
			return ContinuationNone, fmt.Errorf(
				"clear stale enrollment marker: %w", err,
			)
		}

		if inspection.SecureBoot {
			_ = run(
				ctx,
				"sudo", "rm", "-f", "--",
				FinalMarkerPath,
			)
			return ContinuationNone, nil
		}

		if err := run(
			ctx,
			"sudo", "install", "-m", "0600",
			"/dev/null", FinalMarkerPath,
		); err != nil {
			return ContinuationNone, fmt.Errorf(
				"arm final Secure Boot verification: %w", err,
			)
		}

		return ContinuationEnable, nil

	case StateOEMFactoryDerived:
		if err := RecordOwnership(ctx, "pending-enrollment"); err != nil {
			return ContinuationNone, err
		}

	case StatePendingEnrollment:
		if err := RecordOwnership(ctx, "pending-enrollment"); err != nil {
			return ContinuationNone, err
		}

	default:
		return ContinuationNone, fmt.Errorf(
			"refusing to arm Secure Boot enrollment from state %q: %s",
			inspection.State,
			inspection.Description,
		)
	}

	if err := run(
		ctx,
		"sudo", "rm", "-f", "--",
		FinalMarkerPath,
	); err != nil {
		return ContinuationNone, fmt.Errorf(
			"clear stale final Secure Boot marker: %w", err,
		)
	}

	if err := run(
		ctx,
		"sudo", "install", "-m", "0600",
		"/dev/null", EnrollmentMarkerPath,
	); err != nil {
		return ContinuationNone, fmt.Errorf(
			"arm firmware enrollment: %w", err,
		)
	}

	return ContinuationEnroll, nil
}

func verifyBootArtifacts(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "sudo", "sbctl", "verify")
	cmd.Stdin = os.Stdin

	out, err := cmd.CombinedOutput()

	if len(out) > 0 {
		_, _ = os.Stdout.Write(out)
	}

	if err == nil {
		return nil
	}

	allowedUnsignedKernel := false

	for _, raw := range strings.Split(string(out), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || !strings.Contains(line, "is not signed") {
			continue
		}

		path := ""
		for _, field := range strings.Fields(line) {
			if strings.HasPrefix(field, "/boot/") {
				path = field
				break
			}
		}

		if strings.HasPrefix(path, "/boot/EFI/nixos/kernel-") &&
			strings.HasSuffix(path, ".efi") {
			allowedUnsignedKernel = true
			continue
		}

		return fmt.Errorf(
			"unexpected unsigned Secure Boot artifact: %s",
			line,
		)
	}

	if allowedUnsignedKernel {
		return nil
	}

	return fmt.Errorf("verify signed Secure Boot artifacts: %w", err)
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
