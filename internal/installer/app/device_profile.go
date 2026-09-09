package app

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/bakanura/gjallarOS/internal/installer/config"
	"github.com/bakanura/gjallarOS/internal/installer/discovery"
	"github.com/bakanura/gjallarOS/internal/installer/oddc"
)

func resolveDeviceProfile(
	repo string,
	hardware discovery.Hardware,
) (oddc.Resolved, error) {
	source := oddc.EmbeddedSource{
		Root:       filepath.Join(repo, "oddc"),
		Repository: "embedded:oddc",
	}

	resolved, err := source.Resolve(discovery.ODDCIdentity(hardware))
	if err != nil {
		if errors.Is(err, oddc.ErrNoMatch) && hardware.FormFactor != "laptop" {
			return oddc.Resolved{}, nil
		}
		return oddc.Resolved{}, fmt.Errorf("resolve oddc device profile: %w", err)
	}

	return resolved, nil
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
