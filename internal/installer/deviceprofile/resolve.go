package deviceprofile

import (
	"errors"
	"fmt"

	"github.com/JadeOpenServices/gjallarOS/internal/installer/discovery"
	"github.com/JadeOpenServices/gjallarOS/internal/installer/oddc"
)

func Resolve(
	source oddc.DeviceSource,
	hardware discovery.Hardware,
) (oddc.Resolved, error) {
	resolved, err := source.Resolve(
		discovery.ODDCIdentity(hardware),
	)

	return finishResolution(source, resolved, err)
}

// ResolveWithHost resolves the same immutable device source while applying
// machine-local host overlays through an explicit optional source capability.
// The overlays never become part of source metadata.
func ResolveWithHost(
	source oddc.DeviceSource,
	hardware discovery.Hardware,
	host []oddc.HostOverlay,
) (oddc.Resolved, error) {
	if len(host) == 0 {
		return Resolve(source, hardware)
	}

	hostSource, ok := source.(oddc.HostOverlayDeviceSource)
	if !ok {
		return oddc.Resolved{}, fmt.Errorf(
			"ODDC source %T does not support host overlays",
			source,
		)
	}

	resolved, err := hostSource.ResolveWithHost(
		discovery.ODDCIdentity(hardware),
		host,
	)

	return finishResolution(source, resolved, err)
}

func finishResolution(
	source oddc.DeviceSource,
	resolved oddc.Resolved,
	err error,
) (oddc.Resolved, error) {
	if err != nil {
		if errors.Is(err, oddc.ErrNoMatch) {
			return oddc.Resolved{
				Source: source.Metadata(),
			}, nil
		}

		return oddc.Resolved{}, fmt.Errorf(
			"resolve canonical ODDC model: %w",
			err,
		)
	}

	return resolved, nil
}
