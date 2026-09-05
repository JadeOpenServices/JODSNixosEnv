package discovery

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectHardware(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "class", "dmi", "id")
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(path, "chassis_type"), []byte("3\n"), 0644)
	os.WriteFile(filepath.Join(path, "product_name"), []byte("Framework Laptop 13\n"), 0644)
	got := DetectHardware(root)
	if got.FormFactor != "laptop" || got.LaptopVendor != "framework" {
		t.Fatalf("%+v", got)
	}
}

func TestDetectHardwareFindsDirectMultitouchScreen(t *testing.T) {
	root := t.TempDir()
	device := filepath.Join(root, "class", "input", "event4", "device")
	if err := os.MkdirAll(filepath.Join(device, "capabilities"), 0755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(device, "name"), []byte("ELAN Touch Device\n"), 0644)
	os.WriteFile(filepath.Join(device, "properties"), []byte("2\n"), 0644)
	os.WriteFile(filepath.Join(device, "capabilities", "abs"), []byte("60000000000000\n"), 0644)
	if got := DetectHardware(root); !got.Touchscreen {
		t.Fatalf("touchscreen not detected: %+v", got)
	}
}

func TestDetectHardwareDoesNotTreatTouchpadAsTouchscreen(t *testing.T) {
	root := t.TempDir()
	device := filepath.Join(root, "class", "input", "event7", "device")
	if err := os.MkdirAll(filepath.Join(device, "capabilities"), 0755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(device, "name"), []byte("Touchpad\n"), 0644)
	os.WriteFile(filepath.Join(device, "properties"), []byte("5\n"), 0644)
	os.WriteFile(filepath.Join(device, "capabilities", "abs"), []byte("60000000000000\n"), 0644)
	if got := DetectHardware(root); got.Touchscreen {
		t.Fatalf("touchpad detected as touchscreen: %+v", got)
	}
}

func TestDetectHardwareFindsPenTablet(t *testing.T) {
	root := t.TempDir()
	device := filepath.Join(root, "class", "input", "event9", "device")
	if err := os.MkdirAll(filepath.Join(device, "capabilities"), 0755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(device, "name"), []byte("Wacom Tablet\n"), 0644)
	os.WriteFile(filepath.Join(device, "capabilities", "key"), []byte("1"+strings.Repeat("0", 80)+"\n"), 0644)
	os.WriteFile(filepath.Join(device, "capabilities", "abs"), []byte("3\n"), 0644)
	if got := DetectHardware(root); !got.PenTablet {
		t.Fatalf("pen tablet not detected: %+v", got)
	}
}
