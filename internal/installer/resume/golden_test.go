package resume

import (
	"os"
	"testing"
)

// tests/nix/installer-resume.nix evaluates this copy of the wrapper module.
func TestModuleGolden(t *testing.T) {
	const golden = "../../../tests/nix/installer-resume-wrapper.nix"

	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(golden, []byte(Module()), 0644); err != nil {
			t.Fatal(err)
		}
	}

	data, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != Module() {
		t.Fatalf("%s is stale; rerun with UPDATE_GOLDEN=1", golden)
	}
}
