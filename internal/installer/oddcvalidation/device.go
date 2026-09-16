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

	check := func(
		name string,
		recorded string,
		current string,
	) {
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

	check(
		"sysVendor",
		user.DeviceSysVendor,
		hardware.SysVendor,
	)
	check(
		"productName",
		user.DeviceProductName,
		hardware.ProductName,
	)
	check(
		"productVersion",
		user.DeviceProductVersion,
		hardware.ProductVersion,
	)
	check(
		"boardVendor",
		user.DeviceBoardVendor,
		hardware.BoardVendor,
	)
	check(
		"boardName",
		user.DeviceBoardName,
		hardware.BoardName,
	)
	check(
		"boardVersion",
		user.DeviceBoardVersion,
		hardware.BoardVersion,
	)

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
	if user.ODDCModel != resolved.ModelID {
		return Result{
			Gate:   GateProfilePropagation,
			Passed: false,
			Details: fmt.Sprintf(
				"oddcModel=%q resolved=%q",
				user.ODDCModel,
				resolved.ModelID,
			),
		}
	}

	return Result{
		Gate:   GateProfilePropagation,
		Passed: true,
	}
}
