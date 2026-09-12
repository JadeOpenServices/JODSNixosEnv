package app

import (
	"fmt"
	"path/filepath"

	"github.com/bakanura/gjallarOS/internal/installer/deviceprofilecache"
	"github.com/bakanura/gjallarOS/internal/installer/discovery"
	"github.com/bakanura/gjallarOS/internal/installer/oddc"
	"github.com/bakanura/gjallarOS/internal/installer/release"
	"github.com/bakanura/gjallarOS/internal/installer/sourcerevision"
)

func materializeDeviceProfileCapsule(
	repo string,
	destination string,
	hardware discovery.Hardware,
	resolved oddc.Resolved,
	recovery bool,
	generationReason string,
) error {
	revision, err := sourcerevision.Resolve(repo, recovery)
	if err != nil {
		return fmt.Errorf(
			"resolve device-profile source revision: %w",
			err,
		)
	}

	pinnedRelease, err := release.Expected(repo)
	if err != nil {
		return fmt.Errorf(
			"resolve pinned NixOS release for device profile: %w",
			err,
		)
	}

	source := oddc.EmbeddedSource{
		Root:       filepath.Join(repo, "oddc"),
		Repository: resolved.Source.Repository,
		Revision:   resolved.Source.Revision,
		Integrity:  resolved.Source.Integrity,
	}

	if _, err := deviceprofilecache.Materialize(
		deviceprofilecache.MaterializeInput{
			Destination:       destination,
			Identity:          discovery.ODDCIdentity(hardware),
			Resolved:          resolved,
			Source:            source,
			GjallarOSRevision: revision,
			NixOSRelease:      pinnedRelease,
			GenerationReason:  generationReason,
		},
	); err != nil {
		return fmt.Errorf(
			"materialize device-profile capsule: %w",
			err,
		)
	}

	return nil
}
