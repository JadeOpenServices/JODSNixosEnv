package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	portable "github.com/JadeOpenServices/oddc/pkg/oddc"

	"github.com/JadeOpenServices/gjallarOS/internal/installer/oddcdraft"
)

// stageODDCDraft drafts an ODDC model for a machine ODDC does not match and
// installs it next to the device profile (…/var/lib/gjallarOS/oddc-draft),
// for the user to refine and contribute. It is never deployed. A failed
// draft only warns: the install itself does not depend on it.
func stageODDCDraft(ctx context.Context, profileDestination string, out io.Writer) {
	destination := filepath.Join(filepath.Dir(profileDestination), "oddc-draft")
	if err := writeODDCDraft(ctx, destination, out); err != nil {
		fmt.Fprintf(out, "WARNING: no ODDC draft for this machine: %v\n", err)
	}
}

func writeODDCDraft(ctx context.Context, destination string, out io.Writer) error {
	facts := portable.ReadFacts("/sys")

	stage, err := os.MkdirTemp("", "gjallar-oddc-draft-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	staged := filepath.Join(stage, filepath.Base(destination))

	id, err := oddcdraft.Write(oddcdraft.Catalog, facts, staged)
	if err != nil {
		return err
	}
	if err := installTreePrivileged(ctx, staged, destination); err != nil {
		return err
	}
	fmt.Fprintf(out, "ODDC draft %s written to %s (not deployed)\n", id, destination)
	return nil
}
