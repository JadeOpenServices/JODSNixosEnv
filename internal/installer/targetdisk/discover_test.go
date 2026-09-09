package targetdisk

import (
	"os"
	"strings"
	"testing"
)

func TestDiscoveryUsesExistingValidatorAsSafetyGate(t *testing.T) {
	source, err := os.ReadFile("discover.go")
	if err != nil {
		t.Fatal(err)
	}

	body := string(source)

	for _, want := range []string{
		`"PATH,TYPE"`,
		`device.Type`,
		`!= "disk"`,
		`Validate(`,
		`MinSizeBytes: minSizeBytes`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("target discovery lost safety contract %q", want)
		}
	}
}

func TestDiscoveryDoesNotPerformDiskMutation(t *testing.T) {
	source, err := os.ReadFile("discover.go")
	if err != nil {
		t.Fatal(err)
	}

	body := string(source)

	for _, forbidden := range []string{
		"sgdisk",
		"parted",
		"wipefs",
		"mkfs",
		"cryptsetup",
		"mount ",
		"dd ",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("discovery contains forbidden mutation primitive %q", forbidden)
		}
	}
}
