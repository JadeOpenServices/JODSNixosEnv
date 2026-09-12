package oddcvalidation

import (
	"fmt"

	"github.com/bakanura/gjallarOS/internal/installer/oddc"
	"github.com/bakanura/gjallarOS/internal/installer/secureboot"
)

func secureBootPolicyResult(resolved oddc.Resolved) Result {
	effective, err := oddc.ResolveSecureBootFirmwarePolicy(resolved)
	if err != nil {
		return Result{
			Gate:    GateSecureBootPolicy,
			Details: fmt.Sprintf("resolve Secure Boot policy: %v", err),
		}
	}

	if effective.SourceLayer == "" {
		return Result{
			Gate:    GateSecureBootPolicy,
			Details: "no validated Secure Boot firmware policy is defined for the resolved device profile",
		}
	}

	if !effective.Policy.Supported {
		if effective.Policy.UnsupportedReason == "" {
			return Result{
				Gate:    GateSecureBootPolicy,
				Details: "unsupported Secure Boot policy has no reason",
			}
		}

		return Result{
			Gate:   GateSecureBootPolicy,
			Passed: true,
			Details: "Secure Boot explicitly unsupported: " +
				effective.Policy.UnsupportedReason,
		}
	}

	snapshot, err := secureboot.SnapshotFirmwarePolicy(
		resolved.Device.ID,
		effective,
	)
	if err != nil {
		return Result{
			Gate:    GateSecureBootPolicy,
			Details: fmt.Sprintf("snapshot Secure Boot policy: %v", err),
		}
	}

	if err := secureboot.ValidateFirmwarePolicySnapshot(snapshot); err != nil {
		return Result{
			Gate:    GateSecureBootPolicy,
			Details: fmt.Sprintf("validate Secure Boot policy: %v", err),
		}
	}

	return Result{
		Gate:   GateSecureBootPolicy,
		Passed: true,
	}
}
