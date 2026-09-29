package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bakanura/gjallarOS/internal/installer/deviceprofilecache"
	"github.com/bakanura/gjallarOS/internal/installer/discovery"
	"github.com/bakanura/gjallarOS/internal/installer/oddc"
	"github.com/bakanura/gjallarOS/internal/installer/release"
	"github.com/bakanura/gjallarOS/internal/installer/sourcerevision"
)

func materializeODDCCapsule(
	ctx context.Context,
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

	// The destination is root-owned (/mnt/var/... or /var/...): build the
	// capsule in a private user directory and install it via sudo.
	stage, err := os.MkdirTemp("", "gjallar-device-profile-*")
	if err != nil {
		return fmt.Errorf("create device-profile staging directory: %w", err)
	}
	defer os.RemoveAll(stage)
	staged := filepath.Join(stage, filepath.Base(destination))

	if _, err := deviceprofilecache.Materialize(
		deviceprofilecache.MaterializeInput{
			Destination:       staged,
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

	if _, err := deviceprofilecache.Verify(staged); err != nil {
		return fmt.Errorf("verify staged device-profile capsule: %w", err)
	}

	return installTreePrivileged(ctx, staged, destination)
}
