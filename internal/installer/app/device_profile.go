package app

import (
	"context"
	"fmt"

	"github.com/bakanura/gjallarOS/internal/installer/deviceprofile"
	"github.com/bakanura/gjallarOS/internal/installer/discovery"
	"github.com/bakanura/gjallarOS/internal/installer/oddc"
	"github.com/bakanura/gjallarOS/internal/installer/oddchost"
)

var (
	loadODDCHostOverlay           = oddchost.LoadPrivileged
	saveODDCHostOverlayPrivileged = oddchost.SavePrivileged
)

func currentODDCSource(
	repo string,
	revision string,
) oddc.DeviceSource {
	return deviceprofile.CurrentEmbeddedSource(
		repo,
		revision,
	)
}

func resolveODDCModel(
	repo string,
	revision string,
	hardware discovery.Hardware,
) (oddc.Resolved, error) {
	return resolveODDCModelFromSource(
		currentODDCSource(repo, revision),
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

func resolveODDCModelFromSourceWithHost(
	source oddc.DeviceSource,
	hardware discovery.Hardware,
	host *oddc.HostOverlay,
) (oddc.Resolved, error) {
	if host == nil {
		return resolveODDCModelFromSource(
			source,
			hardware,
		)
	}

	return deviceprofile.ResolveWithHost(
		source,
		hardware,
		[]oddc.HostOverlay{*host},
	)
}

// resolveODDCModelWithHost applies durable machine-local ODDC state only after
// the canonical model/source has already been selected. Host state therefore
// cannot participate in model matching or source identity.
func resolveODDCModelWithHost(
	ctx context.Context,
	installedRoot string,
	source oddc.DeviceSource,
	hardware discovery.Hardware,
	base oddc.Resolved,
) (
	oddc.Resolved,
	*oddc.HostOverlay,
	error,
) {
	if base.ModelID == "" {
		return base, nil, nil
	}

	path, err := oddchost.Path(installedRoot)
	if err != nil {
		return oddc.Resolved{}, nil, err
	}

	host, exists, err := loadODDCHostOverlay(
		ctx,
		path,
		base.ModelID,
	)
	if err != nil {
		return oddc.Resolved{}, nil, fmt.Errorf(
			"load machine-local ODDC host overlay: %w",
			err,
		)
	}
	if !exists {
		return base, nil, nil
	}

	resolved, err := resolveODDCModelFromSourceWithHost(
		source,
		hardware,
		&host,
	)
	if err != nil {
		return oddc.Resolved{}, nil, fmt.Errorf(
			"resolve ODDC model with machine-local host overlay: %w",
			err,
		)
	}

	if resolved.ModelID != base.ModelID {
		return oddc.Resolved{}, nil, fmt.Errorf(
			"ODDC host overlay changed resolved model from %q to %q",
			base.ModelID,
			resolved.ModelID,
		)
	}

	return resolved, &host, nil
}

// Machine-local hardware decisions belong to one already-bound installed
// machine. Never inherit them into a fresh install or across device rebind.
func shouldApplyODDCHostOverlay(
	persistentInstalledHost bool,
	needsDeviceRebind bool,
) bool {
	return persistentInstalledHost && !needsDeviceRebind
}

// hostOverlayForCommit returns the complete host overlay that should become
// durable after the enclosing machine deployment succeeds.
//
// A hardware rebind always establishes fresh host state for the newly bound
// model. This intentionally replaces any stale overlay belonging to the old
// machine/model instead of migrating it implicitly.
func hostOverlayForCommit(
	active *oddc.HostOverlay,
	changed bool,
	needsDeviceRebind bool,
	modelID string,
) (*oddc.HostOverlay, error) {
	if needsDeviceRebind {
		if active != nil {
			if active.TargetModel != modelID {
				return nil, fmt.Errorf(
					"staged ODDC host overlay targets %q, expected rebound model %q",
					active.TargetModel,
					modelID,
				)
			}

			return active, nil
		}

		replacement, err := oddchost.New(modelID)
		if err != nil {
			return nil, err
		}

		return &replacement, nil
	}

	if !changed {
		return nil, nil
	}

	if active == nil {
		return nil, fmt.Errorf(
			"ODDC host overlay changed without staged machine-local state",
		)
	}

	if active.TargetModel != modelID {
		return nil, fmt.Errorf(
			"staged ODDC host overlay targets %q, expected model %q",
			active.TargetModel,
			modelID,
		)
	}

	return active, nil
}

func saveODDCHostOverlay(
	ctx context.Context,
	installedRoot string,
	overlay *oddc.HostOverlay,
) error {
	if overlay == nil {
		return nil
	}

	path, err := oddchost.Path(installedRoot)
	if err != nil {
		return err
	}

	if err := saveODDCHostOverlayPrivileged(
		ctx,
		path,
		*overlay,
	); err != nil {
		return fmt.Errorf(
			"commit machine-local ODDC host overlay: %w",
			err,
		)
	}

	return nil
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
