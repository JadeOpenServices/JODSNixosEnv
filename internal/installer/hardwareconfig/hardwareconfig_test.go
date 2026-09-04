package hardwareconfig

import (
	"path/filepath"
	"testing"
)

func TestValidateTarget(t *testing.T) {
	root := t.TempDir()
	valid := filepath.Join(root, "profiles", "laptop", "hardware-configuration.nix")
	if got, err := ValidateTarget(root, valid); err != nil || got != valid {
		t.Fatalf("got %q, %v", got, err)
	}
	for _, invalid := range []string{filepath.Join(root, "hardware-configuration.nix"), filepath.Join(root, "..", "hardware-configuration.nix"), filepath.Join(root, "profiles", "x", "other.nix")} {
		if _, err := ValidateTarget(root, invalid); err == nil {
			t.Fatalf("accepted invalid target %s", invalid)
		}
	}
}
