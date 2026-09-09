package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func writeHardwareFixture(t *testing.T, root, relative, contents string) {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestDetectHardwareDesktopChassis(t *testing.T) {
	root := t.TempDir()
	writeHardwareFixture(t, root, "class/dmi/id/chassis_type", "3\n")

	got := DetectHardware(root)
	if got.FormFactor != "desktop" {
		t.Fatalf("form factor = %q, want desktop", got.FormFactor)
	}
	if got.LaptopVendor != "" {
		t.Fatalf("desktop retained laptop vendor %q", got.LaptopVendor)
	}
}

func TestDetectHardwareBatteryLaptop(t *testing.T) {
	root := t.TempDir()
	writeHardwareFixture(t, root, "class/power_supply/BAT0/status", "Full\n")

	got := DetectHardware(root)
	if got.FormFactor != "laptop" {
		t.Fatalf("form factor = %q, want laptop", got.FormFactor)
	}
	if got.LaptopVendor != "generic" {
		t.Fatalf("laptop vendor = %q, want generic", got.LaptopVendor)
	}
}

func TestDetectHardwarePortableChassis(t *testing.T) {
	for _, chassis := range []string{"8", "9", "10", "14", "30", "31", "32"} {
		t.Run(chassis, func(t *testing.T) {
			root := t.TempDir()
			writeHardwareFixture(t, root, "class/dmi/id/chassis_type", chassis+"\n")

			got := DetectHardware(root)
			if got.FormFactor != "laptop" {
				t.Fatalf("chassis %s form factor = %q, want laptop", chassis, got.FormFactor)
			}
		})
	}
}

func TestDetectHardwareUnknownRemainsUnknown(t *testing.T) {
	got := DetectHardware(t.TempDir())
	if got.FormFactor != "" {
		t.Fatalf("form factor = %q, want unknown", got.FormFactor)
	}
	if got.LaptopVendor != "" {
		t.Fatalf("laptop vendor = %q, want empty", got.LaptopVendor)
	}
}

func TestDetectHardwareFrameworkOverridesUnknown(t *testing.T) {
	root := t.TempDir()
	writeHardwareFixture(t, root, "class/dmi/id/product_name", "Framework Laptop 13\n")

	got := DetectHardware(root)
	if got.FormFactor != "laptop" || got.LaptopVendor != "framework" {
		t.Fatalf("got %+v, want Framework laptop", got)
	}
}
