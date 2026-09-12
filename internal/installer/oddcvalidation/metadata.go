package oddcvalidation

import (
	"fmt"
	"strings"
	"time"

	"github.com/bakanura/gjallarOS/internal/installer/oddc"
)

func ValidationMetadata(
	report Report,
	device DeviceContext,
	nixOSRelease string,
	now time.Time,
) (oddc.Validation, error) {
	if err := report.Complete(); err != nil {
		return oddc.Validation{}, err
	}

	if strings.TrimSpace(nixOSRelease) == "" {
		return oddc.Validation{}, fmt.Errorf("validated NixOS release is empty")
	}
	if strings.TrimSpace(device.Revision) == "" {
		return oddc.Validation{}, fmt.Errorf("validated GjallarOS revision is empty")
	}
	if strings.TrimSpace(device.Resolved.Device.ID) == "" {
		return oddc.Validation{}, fmt.Errorf("validated ODDC device ID is empty")
	}
	if strings.TrimSpace(device.Resolved.Source.Revision) == "" {
		return oddc.Validation{}, fmt.Errorf("validated ODDC revision is empty")
	}

	return oddc.Validation{
		LastValidatedNixOS:             nixOSRelease,
		LastValidatedGjallarOSRevision: device.Revision,
		LastValidatedDeviceID:          device.Resolved.Device.ID,
		LastValidatedODDCRevision:      device.Resolved.Source.Revision,
		LastValidatedAt:                now.UTC().Format(time.RFC3339),
	}, nil
}
