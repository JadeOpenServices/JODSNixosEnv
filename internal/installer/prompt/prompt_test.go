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

func TestGTKConfirmationWidthBoundaries(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    int
	}{
		{
			name:    "short yes no remains compact",
			message: "Continue?",
			want:    420,
		},
		{
			name:    "compact upper boundary",
			message: strings.Repeat("a", 90),
			want:    420,
		},
		{
			name:    "normal by total length",
			message: strings.Repeat("a", 91),
			want:    520,
		},
		{
			name:    "normal upper boundary",
			message: strings.Repeat("a", 180) + "\n\n" + strings.Repeat("b", 60),
			want:    520,
		},
		{
			name:    "long by paragraph length",
			message: strings.Repeat("a", 181),
			want:    680,
		},
		{
			name:    "long by total length",
			message: strings.Repeat("a", 120) + "\n\n" + strings.Repeat("b", 121),
			want:    680,
		},
		{
			name:    "long upper boundary",
			message: strings.Repeat("a", 300) + "\n\n" + strings.Repeat("b", 300),
			want:    680,
		},
		{
			name:    "very long by total length",
			message: strings.Repeat("a", 300) + "\n\n" + strings.Repeat("b", 301),
			want:    760,
		},
		{
			name:    "very long by paragraph length",
			message: strings.Repeat("a", 401),
			want:    760,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := gtkConfirmationWidth(tt.message); got != tt.want {
				t.Fatalf(
					"gtkConfirmationWidth() = %d, want %d",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestGTKConfirmationWidthCountsRunesNotBytes(t *testing.T) {
	message := strings.Repeat("ä", 90)

	if got := gtkConfirmationWidth(message); got != 420 {
		t.Fatalf(
			"90 Unicode runes produced width %d, want compact 420",
			got,
		)
	}

	if len(message) <= 90 {
		t.Fatal("test fixture must contain more bytes than runes")
	}
}

func TestGTKConfirmationWidthLongRecoveryCompatibilityWarning(t *testing.T) {
	message := `The existing installation cannot use the requested trusted recovery layout with its current filesystem configuration.

GjallarOS recovery requires compatible Btrfs state so the recovery environment can repair or reconstruct the installed system safely. Continuing without that compatibility changes which recovery guarantees are available.

Do you want to continue without the incompatible recovery configuration?`

	if got := gtkConfirmationWidth(message); got != 680 {
		t.Fatalf(
			"long recovery compatibility warning width = %d, want 680",
			got,
		)
	}
}

func TestGTKConfirmationWidthVeryLongDeviceCompatibilityWarning(t *testing.T) {
	message := `The detected device does not currently have a validated firmware ownership-transfer procedure for this security operation.

GjallarOS uses the resolved ODDC device profile to determine whether firmware behavior has been validated for the exact device family. The device policy may describe supported firmware actions, but it cannot execute commands or bypass the trusted installer boundary.

Continuing with an unsupported firmware procedure could leave the machine unable to boot, replace firmware-owned Secure Boot material, or produce a configuration that cannot be recovered automatically.

This installation can continue only through a supported compatibility path. Review the detected device and security settings before proceeding.`

	if got := gtkConfirmationWidth(message); got != 760 {
		t.Fatalf(
			"very long device compatibility warning width = %d, want 760",
			got,
		)
	}
}

func TestNewWithInjectedIODisablesGTK(t *testing.T) {
	var out bytes.Buffer

	ui := New(strings.NewReader("yes\n"), &out)

	if ui.GTK {
		t.Fatal("injected prompt I/O unexpectedly enabled GTK")
	}
}
