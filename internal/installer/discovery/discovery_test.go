package discovery

import (
	"os"
	"path/filepath"
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
