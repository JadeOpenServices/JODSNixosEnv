package oddcvalidation

import (
	"fmt"
	"strings"

	"github.com/bakanura/gjallarOS/internal/installer/config"
	"github.com/bakanura/gjallarOS/internal/installer/discovery"
	"github.com/bakanura/gjallarOS/internal/installer/oddc"
)

func hardwareIdentityResult(
	user config.User,
	hardware discovery.Hardware,
) Result {
	mismatches := []string{}

	check := func(name, recorded, current string) {
		if recorded != current {
			mismatches = append(
				mismatches,
				fmt.Sprintf(
					"%s recorded=%q current=%q",
					name,
					recorded,
					current,
				),
			)
		}
	}

	check("sysVendor", user.DeviceSysVendor, hardware.SysVendor)
	check("productName", user.DeviceProductName, hardware.ProductName)
	check("productVersion", user.DeviceProductVersion, hardware.ProductVersion)
	check("boardVendor", user.DeviceBoardVendor, hardware.BoardVendor)
	check("boardName", user.DeviceBoardName, hardware.BoardName)
	check("boardVersion", user.DeviceBoardVersion, hardware.BoardVersion)

	if len(mismatches) != 0 {
		return Result{
			Gate:    GateHardwareIdentity,
			Passed:  false,
			Details: strings.Join(mismatches, "; "),
		}
	}

	return Result{
		Gate:   GateHardwareIdentity,
		Passed: true,
	}
}

func profilePropagationResult(
	user config.User,
	resolved oddc.Resolved,
) Result {
	if user.DeviceProfile != resolved.Device.ID {
		return Result{
			Gate:   GateProfilePropagation,
			Passed: false,
			Details: fmt.Sprintf(
				"deviceProfile=%q resolved=%q",
				user.DeviceProfile,
				resolved.Device.ID,
			),
		}
	}

	expected := make([]string, 0, len(resolved.Inheritance))
	for _, layer := range resolved.Inheritance {
		expected = append(expected, layer.ID)
	}

	if len(user.DeviceLayers) != len(expected) {
		return Result{
			Gate:   GateProfilePropagation,
			Passed: false,
			Details: fmt.Sprintf(
				"deviceLayers=%v resolved=%v",
				user.DeviceLayers,
				expected,
			),
		}
	}

	for i := range expected {
		if user.DeviceLayers[i] != expected[i] {
			return Result{
				Gate:   GateProfilePropagation,
				Passed: false,
				Details: fmt.Sprintf(
					"deviceLayers=%v resolved=%v",
					user.DeviceLayers,
					expected,
				),
			}
		}
	}

	return Result{
		Gate:   GateProfilePropagation,
		Passed: true,
	}
}
