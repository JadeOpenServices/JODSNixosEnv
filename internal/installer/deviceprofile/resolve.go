package deviceprofile

import (
	"errors"
	"fmt"

	"github.com/bakanura/gjallarOS/internal/installer/discovery"
	"github.com/bakanura/gjallarOS/internal/installer/oddc"
)

func Resolve(
	source oddc.DeviceSource,
	hardware discovery.Hardware,
) (oddc.Resolved, error) {
	resolved, err := source.Resolve(discovery.ODDCIdentity(hardware))
	if err != nil {
		if errors.Is(err, oddc.ErrNoMatch) && hardware.FormFactor != "laptop" {
			return oddc.Resolved{
				Source: source.Metadata(),
			}, nil
		}

		return oddc.Resolved{}, fmt.Errorf(
			"resolve oddc device profile: %w",
			err,
		)
	}

	return resolved, nil
}
