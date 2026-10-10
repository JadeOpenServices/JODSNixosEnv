package app

import (
	"os"
	"os/user"
	"testing"
)

func TestDefaultUsernameNeverSuggestsRoot(t *testing.T) {
	if got := defaultUsername("/"); got != "user" {
		t.Fatalf("root-owned checkout suggested %q", got)
	}
	if got := defaultUsername("/does/not/exist"); got != "user" {
		t.Fatalf("missing checkout suggested %q", got)
	}
}

func TestDefaultUsernameIsCheckoutOwner(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("checkout would be root-owned")
	}
	account, err := user.Current()
	if err != nil || account.Username == "nixos" {
		t.Skip("test account has no usable name")
	}
	if got := defaultUsername(t.TempDir()); got != account.Username {
		t.Fatalf("defaultUsername = %q, want %q", got, account.Username)
	}
}
