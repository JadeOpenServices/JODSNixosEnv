package oddcvalidation

import (
	"strings"
	"testing"

	"github.com/bakanura/gjallarOS/internal/installer/config"
	"github.com/bakanura/gjallarOS/internal/installer/discovery"
	"github.com/bakanura/gjallarOS/internal/installer/oddc"
)

func TestHardwareIdentityGate(t *testing.T) {
	hardware := discovery.Hardware{
		SysVendor:      "HP",
		ProductName:    "HP ZBook x2 G4",
		ProductVersion: "A",
		BoardVendor:    "HP",
		BoardName:      "824C",
		BoardVersion:   "43.72",
	}

	user := config.User{
		DeviceSysVendor:      hardware.SysVendor,
		DeviceProductName:    hardware.ProductName,
		DeviceProductVersion: hardware.ProductVersion,
		DeviceBoardVendor:    hardware.BoardVendor,
		DeviceBoardName:      hardware.BoardName,
		DeviceBoardVersion:   hardware.BoardVersion,
	}

	if result := hardwareIdentityResult(user, hardware); !result.Passed {
		t.Fatalf("matching identity rejected: %+v", result)
	}

	user.DeviceBoardName = "wrong"
	if result := hardwareIdentityResult(user, hardware); result.Passed {
		t.Fatal("mismatched identity accepted")
	}
}

func TestProfilePropagationGateUsesCanonicalModel(t *testing.T) {
	const modelID = "model/framework/laptop-13-amd-ryzen-7040"

	resolved := oddc.Resolved{
		ModelID: modelID,
	}

	result := profilePropagationResult(
		config.User{
			ODDCModel: modelID,
		},
		resolved,
	)

	if !result.Passed {
		t.Fatalf(
			"matching canonical model rejected: %+v",
			result,
		)
	}

	result = profilePropagationResult(
		config.User{
			ODDCModel: "model/framework/old-device",
		},
		resolved,
	)

	if result.Passed {
		t.Fatal(
			"canonical model mismatch was accepted",
		)
	}

	if !strings.Contains(
		result.Details,
		"oddcModel=",
	) {
		t.Fatalf(
			"unexpected propagation failure: %q",
			result.Details,
		)
	}
}
