package oddcvalidation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bakanura/gjallarOS/internal/installer/config"
	"github.com/bakanura/gjallarOS/internal/installer/discovery"
)

func TestLoadDeviceContextUsesInjectedInputs(t *testing.T) {
	repo := t.TempDir()

	deviceDir := filepath.Join(
		repo,
		"oddc",
		"devices",
		"laptop",
		"common",
	)

	if err := os.MkdirAll(deviceDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(deviceDir, "device.json"),
		[]byte(`{
		  "schema": 1,
		  "id": "laptop/common",
		  "class": "laptop",
		  "lifecycle": {"status": "supported"},
		  "validation": {}
		}`),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	hardware := discovery.Hardware{
		FormFactor:  "laptop",
		SysVendor:   "Test Vendor",
		ProductName: "Test Laptop",
	}

	user := config.User{
		Hostname:          "gjallarOS",
		DeviceProfile:     "laptop/common",
		DeviceLayers:      []string{"laptop/common"},
		DeviceSysVendor:   hardware.SysVendor,
		DeviceProductName: hardware.ProductName,
	}

	ctx, err := loadDeviceContext(
		repo,
		func(path string) (config.User, error) {
			return user, nil
		},
		func(repo string, recovery bool) (string, error) {
			return "git:test", nil
		},
		func(sysRoot string) discovery.Hardware {
			return hardware
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if ctx.Revision != "git:test" {
		t.Fatalf("Revision=%q", ctx.Revision)
	}

	if ctx.Resolved.Device.ID != "laptop/common" {
		t.Fatalf("Device.ID=%q", ctx.Resolved.Device.ID)
	}

	if ctx.Resolved.Source.Revision != "git:test" {
		t.Fatalf(
			"Source.Revision=%q",
			ctx.Resolved.Source.Revision,
		)
	}

	for _, result := range ctx.ValidationResults() {
		if !result.Passed {
			t.Fatalf("validation result failed: %+v", result)
		}
	}
}
