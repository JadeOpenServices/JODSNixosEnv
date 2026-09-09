package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bakanura/gjallarOS/internal/installer/config"
	"github.com/bakanura/gjallarOS/internal/installer/discovery"
	"github.com/bakanura/gjallarOS/internal/installer/oddc"
)

func TestResolveDeviceProfileFallsBackToLaptopCommon(t *testing.T) {
	repo := t.TempDir()

	device := filepath.Join(repo, "oddc", "devices", "laptop", "common")
	if err := os.MkdirAll(device, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(device, "device.json"),
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

	resolved, err := resolveDeviceProfile(repo, discovery.Hardware{
		FormFactor:  "laptop",
		SysVendor:   "Unknown Vendor",
		ProductName: "Unknown Laptop",
	})
	if err != nil {
		t.Fatal(err)
	}

	if resolved.Device.ID != "laptop/common" {
		t.Fatalf("Device.ID=%q", resolved.Device.ID)
	}
}

func TestResolveDeviceProfileAllowsNoDesktopProfile(t *testing.T) {
	repo := t.TempDir()

	if err := os.MkdirAll(
		filepath.Join(repo, "oddc", "devices"),
		0755,
	); err != nil {
		t.Fatal(err)
	}

	resolved, err := resolveDeviceProfile(repo, discovery.Hardware{
		FormFactor: "desktop",
	})
	if err != nil {
		t.Fatal(err)
	}

	if resolved.Device.ID != "" {
		t.Fatalf("unexpected desktop device profile %q", resolved.Device.ID)
	}
}

func TestPersistDeviceIdentity(t *testing.T) {
	hardware := discovery.Hardware{
		FormFactor:     "laptop",
		SysVendor:      "HP",
		ProductName:    "HP ZBook x2 G4",
		ProductVersion: "A",
		BoardVendor:    "HP",
		BoardName:      "824C",
		BoardVersion:   "KBC Version 43.72",
	}

	repo := t.TempDir()
	device := filepath.Join(repo, "oddc", "devices", "laptop", "common")
	if err := os.MkdirAll(device, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(device, "device.json"),
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

	resolved, err := resolveDeviceProfile(repo, hardware)
	if err != nil {
		t.Fatal(err)
	}

	var user config.User
	persistDeviceIdentity(&user, hardware, resolved)

	if user.DeviceProfile != "laptop/common" {
		t.Fatalf("DeviceProfile=%q", user.DeviceProfile)
	}
	if user.DeviceSysVendor != hardware.SysVendor {
		t.Fatalf("DeviceSysVendor=%q", user.DeviceSysVendor)
	}
	if user.DeviceProductName != hardware.ProductName {
		t.Fatalf("DeviceProductName=%q", user.DeviceProductName)
	}
	if user.DeviceProductVersion != hardware.ProductVersion {
		t.Fatalf("DeviceProductVersion=%q", user.DeviceProductVersion)
	}
	if user.DeviceBoardVendor != hardware.BoardVendor {
		t.Fatalf("DeviceBoardVendor=%q", user.DeviceBoardVendor)
	}
	if user.DeviceBoardName != hardware.BoardName {
		t.Fatalf("DeviceBoardName=%q", user.DeviceBoardName)
	}
	if user.DeviceBoardVersion != hardware.BoardVersion {
		t.Fatalf("DeviceBoardVersion=%q", user.DeviceBoardVersion)
	}
}

func TestPersistDeviceIdentityOverwritesStalePresetValues(t *testing.T) {
	user := config.User{
		DeviceProfile:        "laptop/fake/model",
		DeviceSysVendor:      "Fake",
		DeviceProductName:    "Fake Product",
		DeviceProductVersion: "Fake Version",
		DeviceBoardVendor:    "Fake Board Vendor",
		DeviceBoardName:      "Fake Board",
		DeviceBoardVersion:   "Fake Board Version",
	}

	hardware := discovery.Hardware{
		FormFactor:     "laptop",
		SysVendor:      "HP",
		ProductName:    "HP ZBook x2 G4",
		ProductVersion: "A",
		BoardVendor:    "HP",
		BoardName:      "824C",
		BoardVersion:   "KBC Version 43.72",
	}

	resolved := oddc.Resolved{
		Device: oddc.Manifest{
			ID: "laptop/common",
		},
	}

	persistDeviceIdentity(&user, hardware, resolved)

	if user.DeviceProfile != "laptop/common" {
		t.Fatalf("DeviceProfile=%q", user.DeviceProfile)
	}
	if user.DeviceSysVendor != "HP" {
		t.Fatalf("DeviceSysVendor=%q", user.DeviceSysVendor)
	}
	if user.DeviceProductName != "HP ZBook x2 G4" {
		t.Fatalf("DeviceProductName=%q", user.DeviceProductName)
	}
	if user.DeviceProductVersion != "A" {
		t.Fatalf("DeviceProductVersion=%q", user.DeviceProductVersion)
	}
	if user.DeviceBoardVendor != "HP" {
		t.Fatalf("DeviceBoardVendor=%q", user.DeviceBoardVendor)
	}
	if user.DeviceBoardName != "824C" {
		t.Fatalf("DeviceBoardName=%q", user.DeviceBoardName)
	}
	if user.DeviceBoardVersion != "KBC Version 43.72" {
		t.Fatalf("DeviceBoardVersion=%q", user.DeviceBoardVersion)
	}
}

func TestPersistDeviceIdentityPreservesResolvedLayerOrder(t *testing.T) {
	user := config.User{}

	resolved := oddc.Resolved{
		Device: oddc.Manifest{
			ID: "laptop/hp/zbook-x2-g4",
		},
		Inheritance: []oddc.Manifest{
			{ID: "laptop/common"},
			{ID: "laptop/hp"},
			{ID: "laptop/hp/zbook-x2-g4"},
		},
	}

	persistDeviceIdentity(
		&user,
		discovery.Hardware{FormFactor: "laptop"},
		resolved,
	)

	want := []string{
		"laptop/common",
		"laptop/hp",
		"laptop/hp/zbook-x2-g4",
	}

	if len(user.DeviceLayers) != len(want) {
		t.Fatalf("DeviceLayers=%v", user.DeviceLayers)
	}

	for i := range want {
		if user.DeviceLayers[i] != want[i] {
			t.Fatalf(
				"DeviceLayers[%d]=%q want %q",
				i,
				user.DeviceLayers[i],
				want[i],
			)
		}
	}
}
