package oddcvalidation

import (
	"context"
	"testing"

	"github.com/bakanura/gjallarOS/internal/installer/config"
	"github.com/bakanura/gjallarOS/internal/installer/discovery"
	"github.com/bakanura/gjallarOS/internal/installer/oddc"
)

func TestRunDeviceStageOrdersRebuildIdentityAndPropagation(t *testing.T) {
	device := DeviceContext{
		User: config.User{
			Hostname:             "gjallarOS",
			DeviceProfile:        "laptop/framework",
			DeviceLayers:         []string{"laptop/common", "laptop/framework"},
			DeviceSysVendor:      "Framework",
			DeviceProductName:    "Laptop",
			DeviceProductVersion: "A",
			DeviceBoardVendor:    "Framework",
			DeviceBoardName:      "Board",
			DeviceBoardVersion:   "1",
		},
		Hardware: discovery.Hardware{
			FormFactor:     "laptop",
			SysVendor:      "Framework",
			ProductName:    "Laptop",
			ProductVersion: "A",
			BoardVendor:    "Framework",
			BoardName:      "Board",
			BoardVersion:   "1",
		},
		Resolved: oddc.Resolved{
			Device: oddc.Manifest{
				ID: "laptop/framework",
			},
			Inheritance: []oddc.Manifest{
				{ID: "laptop/common"},
				{ID: "laptop/framework"},
			},
		},
	}

	runner := func(
		ctx context.Context,
		dir string,
		name string,
		args ...string,
	) ([]byte, error) {
		return nil, nil
	}

	report, _, err := runDeviceStage(
		context.Background(),
		"/repo",
		runner,
		func(string) (DeviceContext, error) {
			return device, nil
		},
		func() (string, error) {
			return "/usr/bin/gjallarctl", nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	want := []Gate{
		GateRealMachineRebuild,
		GateHardwareIdentity,
		GateProfilePropagation,
		GateRuntimeGraphics,
		GateRuntimeSensorsTablet,
		GateSecureBootPolicy,
	}

	if len(report.Results) != len(want) {
		t.Fatalf("results=%d want %d", len(report.Results), len(want))
	}

	for i, gate := range want {
		if report.Results[i].Gate != gate {
			t.Fatalf(
				"result[%d].Gate=%q want %q",
				i,
				report.Results[i].Gate,
				gate,
			)
		}
	}
}
