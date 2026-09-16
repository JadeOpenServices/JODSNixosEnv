package app

import (
	"strings"
	"testing"

	"github.com/bakanura/gjallarOS/internal/installer/config"
	"github.com/bakanura/gjallarOS/internal/installer/discovery"
	"github.com/bakanura/gjallarOS/internal/installer/oddc"
)

func TestSecureBootSupportGateAcceptsDetectedSupportedPolicy(t *testing.T) {
	err := validateSecureBootFirmwareSupport(
		true,
		"laptop/framework",
		oddc.EffectiveSecureBootFirmwarePolicy{
			SourceEntity: "vendor/framework",
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
			SourceEntity: "vendor/hp",
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
			SourceEntity: "vendor/hp",
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

func TestPersistDeviceIdentityUsesCanonicalModel(t *testing.T) {
	user := config.User{
		ODDCModel: "model/obsolete/device",
	}

	hardware := discovery.Hardware{
		FormFactor:     "laptop",
		SysVendor:      "Framework",
		ProductName:    "Laptop 13 (AMD Ryzen 7040Series)",
		ProductVersion: "A7",
		BoardVendor:    "Framework",
		BoardName:      "FRANMDCP07",
		BoardVersion:   "A7",
	}

	resolved := oddc.Resolved{
		ModelID: "model/framework/laptop-13-amd-ryzen-7040",
	}

	persistDeviceIdentity(
		&user,
		hardware,
		resolved,
	)

	if user.ODDCModel != resolved.ModelID {
		t.Fatalf(
			"ODDCModel=%q want=%q",
			user.ODDCModel,
			resolved.ModelID,
		)
	}

	if user.DeviceSysVendor != hardware.SysVendor ||
		user.DeviceProductName != hardware.ProductName ||
		user.DeviceBoardName != hardware.BoardName {
		t.Fatalf(
			"hardware identity not persisted: %+v",
			user,
		)
	}
}

func TestODDCModelDrift(t *testing.T) {
	resolved := oddc.Resolved{
		ModelID: "model/hp/zbook-x2-g4",
	}

	if oddcModelDrifted(
		config.User{
			ODDCModel: resolved.ModelID,
		},
		resolved,
	) {
		t.Fatal(
			"matching canonical model was reported as drift",
		)
	}

	if !oddcModelDrifted(
		config.User{
			ODDCModel: "model/hp/old-device",
		},
		resolved,
	) {
		t.Fatal(
			"changed canonical model was not reported as drift",
		)
	}

	if !oddcModelDrifted(
		config.User{},
		resolved,
	) {
		t.Fatal(
			"missing persisted canonical model was not reported as drift",
		)
	}
}

func TestODDCModelDriftAcceptsUnmatchedHardware(t *testing.T) {
	if oddcModelDrifted(
		config.User{},
		oddc.Resolved{},
	) {
		t.Fatal(
			"unmatched hardware with no persisted model was reported as drift",
		)
	}

	if !oddcModelDrifted(
		config.User{
			ODDCModel: "model/removed/device",
		},
		oddc.Resolved{},
	) {
		t.Fatal(
			"removed canonical model was not reported as drift",
		)
	}
}
