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
	resolved, err := source.Resolve(
		discovery.ODDCIdentity(hardware),
	)
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
