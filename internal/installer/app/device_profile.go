package app

import (
	"fmt"

	"github.com/bakanura/gjallarOS/internal/installer/deviceprofile"
	"github.com/bakanura/gjallarOS/internal/installer/discovery"
	"github.com/bakanura/gjallarOS/internal/installer/oddc"
)

func resolveODDCModel(
	repo string,
	revision string,
	hardware discovery.Hardware,
) (oddc.Resolved, error) {
	source := deviceprofile.CurrentEmbeddedSource(
		repo,
		revision,
	)

	return resolveODDCModelFromSource(
		source,
		hardware,
	)
}

func resolveODDCModelFromSource(
	source oddc.DeviceSource,
	hardware discovery.Hardware,
) (oddc.Resolved, error) {
	return deviceprofile.Resolve(
		source,
		hardware,
	)
}

func validateSecureBootFirmwareSupport(
	enabled bool,
	modelID string,
	effective oddc.EffectiveSecureBootFirmwarePolicy,
) error {
	if !enabled {
		return nil
	}

	if effective.Policy.Supported {
		return nil
	}

	model := modelID
	if model == "" {
		model = "unmatched hardware"
	}

	reason := effective.Policy.UnsupportedReason
	if reason == "" {
		reason =
			"no trusted Secure Boot firmware policy is available"
	}

	return fmt.Errorf(
		"Secure Boot ownership transfer is unsupported for ODDC model %q: %s",
		model,
		reason,
	)
}
