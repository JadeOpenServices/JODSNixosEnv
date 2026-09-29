package secureboot

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGeneratePassphrase(t *testing.T) {
	a, err := generatePassphrase()
	if err != nil || len(a) != 64 {
		t.Fatalf("generatePassphrase() = %q, %v", a, err)
	}
	b, err := generatePassphrase()
	if err != nil || a == b {
		t.Fatal("recovery passphrases must be unique")
	}
}

func TestPrivilegedToolResolvesAbsolutePathForSudo(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "sbctl")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	if got := privilegedTool("sbctl"); got != fake {
		t.Fatalf("privilegedTool = %q, want %q (sudo secure_path would not find a nix-shell sbctl)", got, fake)
	}
	t.Setenv("PATH", t.TempDir())
	if got := privilegedTool("sbctl"); got != "sbctl" {
		t.Fatalf("privilegedTool without a resolvable binary = %q", got)
	}
}
