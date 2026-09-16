package oddcvalidation

import (
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

func TestProfilePropagationGate(t *testing.T) {
	resolved := oddc.Resolved{
		Device: oddc.Manifest{
			ID: "laptop/framework",
		},
		Inheritance: []oddc.Manifest{
			{ID: "laptop/common"},
			{ID: "laptop/framework"},
		},
	}

	user := config.User{
		DeviceProfile: "laptop/framework",
		DeviceLayers: []string{
			"laptop/common",
			"laptop/framework",
		},
	}

	if result := profilePropagationResult(user, resolved); !result.Passed {
		t.Fatalf("matching propagation rejected: %+v", result)
	}

	user.DeviceLayers = []string{"laptop/framework"}
	if result := profilePropagationResult(user, resolved); result.Passed {
		t.Fatal("incomplete inheritance propagation accepted")
	}
}
