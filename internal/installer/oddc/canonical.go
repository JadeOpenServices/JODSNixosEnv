package oddc

import (
	"errors"
	"fmt"
	"strings"

	portable "github.com/JadeOpenServices/oddc"
)

type CanonicalResolution = portable.Resolved

func (resolved Resolved) StableDeviceID() string {
	return strings.TrimSpace(resolved.ModelID)
}

func (source EmbeddedSource) resolveCanonical(
	identity Identity,
	host []HostOverlay,
) (Resolved, error) {
	registry, err := portable.LoadRegistry(source.Root)
	if err != nil {
		return Resolved{}, fmt.Errorf(
			"load canonical ODDC registry: %w",
			err,
		)
	}

	modelID, err := registry.MatchModel(
		portable.MachineIdentity{
			FormFactor:     identity.FormFactor,
			SysVendor:      identity.SysVendor,
			ProductName:    identity.ProductName,
			ProductVersion: identity.ProductVersion,
			BoardVendor:    identity.BoardVendor,
			BoardName:      identity.BoardName,
			BoardVersion:   identity.BoardVersion,
		},
	)
	if err != nil {
		if errors.Is(err, portable.ErrNoModelMatch) {
			return Resolved{}, ErrNoMatch
		}
		return Resolved{}, err
	}

	canonical, err := registry.ResolveModel(
		modelID,
		nil,
		host,
	)
	if err != nil {
		return Resolved{}, err
	}

	validations, err := source.loadCanonicalValidations(modelID)
	if err != nil {
		return Resolved{}, err
	}

	return Resolved{
		ModelID:     modelID,
		Canonical:   canonical,
		Validations: validations,
		Source:      source.Metadata(),
	}, nil
}
