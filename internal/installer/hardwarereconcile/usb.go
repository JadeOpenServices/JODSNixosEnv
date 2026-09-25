package hardwarereconcile

import (
	"strings"

	"github.com/bakanura/gjallarOS/internal/hardware/deviceprobe"
	"github.com/bakanura/gjallarOS/internal/usbtrust"
	"github.com/bakanura/gjallarOS/internal/usbtrust/oddcsource"
	"github.com/bakanura/gjallarOS/internal/usbtrust/sysfssource"
)

// InternalUSB compares resolved ODDC internal USB expectations with the current
// kernel/sysfs inventory.
//
// This is discovery/reconciliation only. It has no accepted machine-local
// trust state, makes no persistence decision, and ignores unrelated USB
// devices that do not correspond to an ODDC internal role.
func InternalUSB(
	resolved map[string]any,
	snapshot deviceprobe.Snapshot,
) ([]usbtrust.Finding, error) {
	expected, err := oddcsource.Expected(resolved)
	if err != nil {
		return nil, err
	}

	audit, err := usbtrust.Audit(usbtrust.AuditInput{
		Expected: expected,
		Observed: sysfssource.Observed(snapshot),
	})
	if err != nil {
		return nil, err
	}

	findings := make([]usbtrust.Finding, 0, len(audit.Findings))

	for _, finding := range audit.Findings {
		// ODDC-backed internal findings always carry their canonical role.
		// Findings for unrelated observed USB devices deliberately do not.
		if strings.TrimSpace(finding.Role) == "" {
			continue
		}

		findings = append(findings, finding)
	}

	return findings, nil
}
