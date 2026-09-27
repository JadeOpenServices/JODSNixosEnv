package main

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestPrivilegeFingerprintCommandHasNoPasswordInput(t *testing.T) {
	tty, err := os.CreateTemp(t.TempDir(), "tty")
	if err != nil {
		t.Fatal(err)
	}
	defer tty.Close()

	const authHelper = "/nix/store/test-gjallarctl/bin/gjallar-sudo-auth"

	cmd := privilegeFingerprintCommand(
		context.Background(),
		tty,
		authHelper,
	)

	if cmd.Stdin != nil {
		t.Fatal("fingerprint authentication must not receive password-capable stdin")
	}
	if cmd.Stdout != tty || cmd.Stderr != tty {
		t.Fatal("fingerprint authentication messages must use the controlling TTY")
	}

	if len(cmd.Args) != 2 ||
		cmd.Args[0] != "sudo" ||
		cmd.Args[1] != authHelper {
		t.Fatalf("unexpected fingerprint command: %#v", cmd.Args)
	}

	for _, arg := range cmd.Args {
		if arg == "-A" {
			t.Fatal("fingerprint phase must not use askpass")
		}
	}

	if cmd.SysProcAttr == nil || !cmd.SysProcAttr.Setpgid {
		t.Fatal("fingerprint authentication process group isolation missing")
	}
}

func TestPrivilegePasswordCommandUsesAskpassWithoutStdin(t *testing.T) {
	tty, err := os.CreateTemp(t.TempDir(), "tty")
	if err != nil {
		t.Fatal(err)
	}
	defer tty.Close()

	const askpass = "/nix/store/test-gjallarctl/bin/gjallar-sudo-askpass"
	const authHelper = "/nix/store/test-gjallarctl/bin/gjallar-sudo-auth"

	t.Setenv("SUDO_ASKPASS", "/tmp/untrusted-askpass")

	cmd := privilegePasswordCommand(
		context.Background(),
		tty,
		askpass,
		authHelper,
	)

	if cmd.Stdin != nil {
		t.Fatal("password authentication must not receive password-capable stdin")
	}
	if cmd.Stdout != tty || cmd.Stderr != tty {
		t.Fatal("password authentication messages must use the controlling TTY")
	}

	if len(cmd.Args) != 3 ||
		cmd.Args[0] != "sudo" ||
		cmd.Args[1] != "-A" ||
		cmd.Args[2] != authHelper {
		t.Fatalf("unexpected password command: %#v", cmd.Args)
	}

	if cmd.SysProcAttr == nil || !cmd.SysProcAttr.Setpgid {
		t.Fatal("password authentication process group isolation missing")
	}

	var values []string
	for _, entry := range cmd.Env {
		if strings.HasPrefix(entry, "SUDO_ASKPASS=") {
			values = append(values, entry)
		}
	}

	if len(values) != 1 {
		t.Fatalf("expected exactly one SUDO_ASKPASS entry, got %#v", values)
	}

	if values[0] != "SUDO_ASKPASS="+askpass {
		t.Fatalf("unexpected SUDO_ASKPASS: %q", values[0])
	}
}

func TestEnvironmentWithOverrideReplacesExistingValue(t *testing.T) {
	t.Setenv("SUDO_ASKPASS", "/tmp/old")

	env := environmentWithOverride(
		"SUDO_ASKPASS",
		"/nix/store/new/bin/gjallar-sudo-askpass",
	)

	var matches []string
	for _, entry := range env {
		if strings.HasPrefix(entry, "SUDO_ASKPASS=") {
			matches = append(matches, entry)
		}
	}

	if len(matches) != 1 {
		t.Fatalf("expected one SUDO_ASKPASS entry, got %#v", matches)
	}

	if matches[0] !=
		"SUDO_ASKPASS=/nix/store/new/bin/gjallar-sudo-askpass" {
		t.Fatalf("unexpected override: %q", matches[0])
	}
}
