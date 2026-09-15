package app

import (
	"os"
	"path/filepath"
	"strings"
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

	resolved, err := resolveDeviceProfile(repo, "git:test", discovery.Hardware{
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

	resolved, err := resolveDeviceProfile(repo, "git:test", discovery.Hardware{
		FormFactor: "desktop",
	})
	if err != nil {
		t.Fatal(err)
	}

	if resolved.Device.ID != "" {
		t.Fatalf("unexpected desktop device profile %q", resolved.Device.ID)
	}
	if resolved.Source.Revision != "git:test" {
		t.Fatalf("Source.Revision=%q", resolved.Source.Revision)
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

	resolved, err := resolveDeviceProfile(repo, "git:test", hardware)
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

func TestSecureBootSupportGateAcceptsDetectedSupportedPolicy(t *testing.T) {
	err := validateSecureBootFirmwareSupport(
		true,
		"laptop/framework",
		oddc.EffectiveSecureBootFirmwarePolicy{
			SourceLayer: "laptop/framework",
			Policy: oddc.SecureBootFirmwarePolicy{
				Supported: true,
			},
		},
	)

	if err != nil {
		t.Fatal(err)
	}
}

func TestSecureBootSupportGateRejectsDetectedUnsupportedPolicy(t *testing.T) {
	err := validateSecureBootFirmwareSupport(
		true,
		"laptop/hp/zbook-x2-g4",
		oddc.EffectiveSecureBootFirmwarePolicy{
			SourceLayer: "laptop/hp",
			Policy: oddc.SecureBootFirmwarePolicy{
				Supported:         false,
				SetupModeStrategy: "unsupported",
				UnsupportedReason: "HP transfer is not validated.",
			},
		},
	)

	if err == nil {
		t.Fatal("unsupported detected firmware accepted Secure Boot")
	}

	if !strings.Contains(err.Error(), "laptop/hp/zbook-x2-g4") {
		t.Fatalf("error does not identify detected profile: %v", err)
	}
}

func TestSecureBootSupportGateAllowsDisabledSecureBootOnUnsupportedDevice(t *testing.T) {
	err := validateSecureBootFirmwareSupport(
		false,
		"laptop/hp/zbook-x2-g4",
		oddc.EffectiveSecureBootFirmwarePolicy{
			Policy: oddc.SecureBootFirmwarePolicy{
				Supported:         false,
				SetupModeStrategy: "unsupported",
				UnsupportedReason: "HP transfer is not validated.",
			},
		},
	)

	if err != nil {
		t.Fatal(err)
	}
}

func TestManagedDeviceCannotBypassUnsupportedFirmwarePolicy(t *testing.T) {
	user := config.User{
		EndpointManagedDevice: true,
	}

	normalizeManagementSafety(&user)

	if !user.SecureBootEnable {
		t.Fatal("managed policy did not require Secure Boot")
	}

	err := validateSecureBootFirmwareSupport(
		user.SecureBootEnable,
		"laptop/hp/zbook-x2-g4",
		oddc.EffectiveSecureBootFirmwarePolicy{
			SourceLayer: "laptop/hp",
			Policy: oddc.SecureBootFirmwarePolicy{
				Supported:         false,
				SetupModeStrategy: "unsupported",
				UnsupportedReason: "HP transfer is not validated.",
			},
		},
	)

	if err == nil {
		t.Fatal("managed device bypassed unsupported firmware policy")
	}
}

func TestSecureBootSupportGateRejectsMissingPolicy(t *testing.T) {
	err := validateSecureBootFirmwareSupport(
		true,
		"laptop/test",
		oddc.EffectiveSecureBootFirmwarePolicy{},
	)

	if err == nil {
		t.Fatal("missing Secure Boot firmware policy was accepted")
	}

	if !strings.Contains(err.Error(), "no trusted Secure Boot firmware policy") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeviceProfileDriftDetectsNewExactProfile(t *testing.T) {
	user := config.User{
		DeviceProfile: "laptop/framework",
		DeviceLayers: []string{
			"laptop/common",
			"laptop/framework",
		},
	}

	resolved := oddc.Resolved{
		Device: oddc.Manifest{
			ID: "laptop/framework/laptop-13-amd-ryzen-7040",
		},
		Inheritance: []oddc.Manifest{
			{ID: "laptop/common"},
			{ID: "laptop/framework"},
			{ID: "laptop/framework/laptop-13-amd-ryzen-7040"},
		},
	}

	if !deviceProfileDrifted(user, resolved) {
		t.Fatal("stale vendor-only profile was not detected")
	}
}

func TestDeviceProfileDriftAcceptsResolvedProfile(t *testing.T) {
	user := config.User{
		DeviceProfile: "laptop/hp/zbook-x2-g4",
		DeviceLayers: []string{
			"laptop/common",
			"laptop/hp",
			"laptop/hp/zbook-x2-g4",
		},
	}

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

	if deviceProfileDrifted(user, resolved) {
		t.Fatal("matching resolved profile reported drift")
	}
}

func TestDeviceProfileDriftDetectsGenericFallback(t *testing.T) {
	user := config.User{
		DeviceProfile: "laptop/vendor/old-model",
		DeviceLayers: []string{
			"laptop/common",
			"laptop/vendor",
			"laptop/vendor/old-model",
		},
	}

	resolved := oddc.Resolved{
		Device: oddc.Manifest{
			ID: "laptop/common",
		},
		Inheritance: []oddc.Manifest{
			{ID: "laptop/common"},
		},
	}

	if !deviceProfileDrifted(user, resolved) {
		t.Fatal("fallback from removed concrete profile was not detected")
	}
}
