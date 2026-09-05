package prompt

import (
	"bufio"
	"context"
	"strings"
	"testing"
)

func TestTerminalDefaults(t *testing.T) {
	var out strings.Builder
	u := UI{Reader: bufio.NewReader(strings.NewReader("\n\n")), Out: &out}
	yes, err := u.Confirm(context.Background(), "Continue?", true)
	if err != nil || !yes {
		t.Fatal(yes, err)
	}
	value, err := u.Value(context.Background(), "Name", "default")
	if err != nil || value != "default" {
		t.Fatal(value, err)
	}
}

func TestTerminalSecureBootRecoveryShowsBothRequiredItems(t *testing.T) {
	var out strings.Builder
	u := UI{Reader: bufio.NewReader(strings.NewReader("")), Out: &out}
	archive := "/var/lib/gjallarOS/recovery/secure-boot-keys.tar.enc"
	passphrase := "test-recovery-passphrase"
	if err := u.ShowSecureBootRecovery(context.Background(), archive, passphrase); err != nil {
		t.Fatal(err)
	}
	shown := out.String()
	if !strings.Contains(shown, archive) || !strings.Contains(shown, passphrase) {
		t.Fatalf("recovery dialog omitted required material: %q", shown)
	}
}
