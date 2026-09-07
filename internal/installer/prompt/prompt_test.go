package prompt

import (
	"bufio"
	"bytes"
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

func TestExactConfirmation(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  string
		wantError bool
	}{
		{
			name:     "exact",
			input:    "ERASE /dev/nvme1n1\n",
			expected: "ERASE /dev/nvme1n1",
		},
		{
			name:      "wrong disk",
			input:     "ERASE /dev/nvme0n1\n",
			expected:  "ERASE /dev/nvme1n1",
			wantError: true,
		},
		{
			name:      "case mismatch",
			input:     "erase /dev/nvme1n1\n",
			expected:  "ERASE /dev/nvme1n1",
			wantError: true,
		},
		{
			name:      "leading whitespace",
			input:     " ERASE /dev/nvme1n1\n",
			expected:  "ERASE /dev/nvme1n1",
			wantError: true,
		},
		{
			name:      "trailing whitespace",
			input:     "ERASE /dev/nvme1n1 \n",
			expected:  "ERASE /dev/nvme1n1",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			ui := UI{
				Reader: bufio.NewReader(strings.NewReader(tt.input)),
				Out:    &out,
				GTK:    false,
			}

			err := ui.Exact(
				context.Background(),
				"Type the destructive confirmation",
				tt.expected,
			)

			if tt.wantError && err == nil {
				t.Fatal("invalid confirmation was accepted")
			}
			if !tt.wantError && err != nil {
				t.Fatalf("exact confirmation rejected: %v", err)
			}
		})
	}
}

func TestExactConfirmationRequiresExpectedValue(t *testing.T) {
	var out bytes.Buffer
	ui := UI{
		Reader: bufio.NewReader(strings.NewReader("\n")),
		Out:    &out,
	}

	if err := ui.Exact(context.Background(), "Confirm", ""); err == nil {
		t.Fatal("empty expected confirmation was accepted")
	}
}
