package main

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestPrivilegeAuthenticationCommandUsesAskpassWithoutStdin(t *testing.T) {
	tty, err := os.CreateTemp(t.TempDir(), "tty")
	if err != nil {
		t.Fatal(err)
	}
	defer tty.Close()

	const askpass = "/nix/store/test-gjallarctl/bin/gjallar-sudo-askpass"

	t.Setenv("SUDO_ASKPASS", "/tmp/untrusted-askpass")

	cmd := privilegeAuthenticationCommand(
		context.Background(),
		tty,
		askpass,
	)

	if cmd.Stdin != nil {
		t.Fatal("sudo authentication must not receive password-capable stdin")
	}
	if cmd.Stdout != tty {
		t.Fatal("sudo authentication stdout is not the controlling TTY")
	}
	if cmd.Stderr != tty {
		t.Fatal("sudo authentication stderr is not the controlling TTY")
	}

	if len(cmd.Args) != 3 ||
		cmd.Args[0] != "sudo" ||
		cmd.Args[1] != "-A" ||
		cmd.Args[2] != "-v" {
		t.Fatalf("unexpected sudo authentication command: %#v", cmd.Args)
	}

	if cmd.SysProcAttr == nil || !cmd.SysProcAttr.Setpgid {
		t.Fatal("sudo authentication process group isolation missing")
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
