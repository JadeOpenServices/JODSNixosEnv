package app

import (
	"fmt"

	"github.com/bakanura/gjallarOS/internal/installer/config"
	"github.com/bakanura/gjallarOS/internal/installer/deviceprofile"
	"github.com/bakanura/gjallarOS/internal/installer/discovery"
	"github.com/bakanura/gjallarOS/internal/installer/oddc"
)

func resolveDeviceProfile(
	repo string,
	revision string,
	hardware discovery.Hardware,
) (oddc.Resolved, error) {
	source := deviceprofile.CurrentEmbeddedSource(repo, revision)

	return resolveDeviceProfileFromSource(source, hardware)
}

func resolveDeviceProfileFromSource(
	source oddc.DeviceSource,
	hardware discovery.Hardware,
) (oddc.Resolved, error) {
	return deviceprofile.Resolve(source, hardware)
}

func persistDeviceIdentity(
	user *config.User,
	hardware discovery.Hardware,
	resolved oddc.Resolved,
) {
	user.DeviceProfile = resolved.Device.ID
	user.DeviceLayers = make([]string, 0, len(resolved.Inheritance))
	for _, layer := range resolved.Inheritance {
		user.DeviceLayers = append(user.DeviceLayers, layer.ID)
	}
	user.DeviceSysVendor = hardware.SysVendor
	user.DeviceProductName = hardware.ProductName
	user.DeviceProductVersion = hardware.ProductVersion
	user.DeviceBoardVendor = hardware.BoardVendor
	user.DeviceBoardName = hardware.BoardName
	user.DeviceBoardVersion = hardware.BoardVersion
}

func validateSecureBootFirmwareSupport(
	enabled bool,
	deviceProfile string,
	effective oddc.EffectiveSecureBootFirmwarePolicy,
) error {
	if !enabled {
		return nil
	}

	if effective.Policy.Supported {
		return nil
	}

	profile := deviceProfile
	if profile == "" {
		profile = "unresolved device"
	}

	reason := effective.Policy.UnsupportedReason
	if reason == "" {
		reason = "no trusted Secure Boot firmware policy is available"
	}

	return fmt.Errorf(
		"Secure Boot ownership transfer is unsupported for detected device profile %q: %s",
		profile,
		reason,
	)
}
