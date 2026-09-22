package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func fixture(t *testing.T, root, rel, contents string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestDesktopChassis(t *testing.T) {
	root := t.TempDir()
	fixture(t, root, "class/dmi/id/chassis_type", "3\n")

	got := DetectHardware(root)
	if got.FormFactor != "desktop" {
		t.Fatalf("FormFactor=%q", got.FormFactor)
	}
}

func TestBatteryLaptop(t *testing.T) {
	root := t.TempDir()
	fixture(t, root, "class/power_supply/BAT0/status", "Full\n")

	got := DetectHardware(root)
	if got.FormFactor != "laptop" {
		t.Fatalf("FormFactor=%q", got.FormFactor)
	}
}

func TestUnknownHardware(t *testing.T) {
	got := DetectHardware(t.TempDir())
	if got.FormFactor != "" {
		t.Fatalf("unexpected detection: %+v", got)
	}
}

func TestProductNameDoesNotDefineFormFactor(t *testing.T) {
	root := t.TempDir()
	fixture(t, root, "class/dmi/id/product_name", "Framework Laptop 13\n")

	got := DetectHardware(root)
	if got.FormFactor != "" {
		t.Fatalf("product-specific form-factor policy leaked into discovery: %+v", got)
	}
}
