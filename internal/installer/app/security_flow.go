package app

import "strings"

// Choices when the firmware enforces Secure Boot on a machine ODDC has no
// Secure Boot setup for.
const (
	secureBootOffReboot = "Reboot into firmware setup now to turn Secure Boot off"
	secureBootOffLater  = "Continue; I turn Secure Boot off before the first reboot"
	secureBootOffCancel = "Cancel the installation"
)

func mayOfferSecureBootFallback(
	secureBootEnabled bool,
	managed bool,
	unattended bool,
) bool {
	return secureBootEnabled && !managed && !unattended
}

func tpm2AllowedForSecureBoot(secureBootEnabled bool) bool {
	return secureBootEnabled
}

// tpm2FollowsRequest reports whether the rendered TPM2 unlock setting is the
// user's request instead of a probe of the running system. Managed endpoints
// are TPM2-bound by contract, and a fresh install's LUKS2 root is created by
// rootprovision after this decision, so the live media's mappings say nothing
// about it (e2e-target, 2026-09-29: rendered false, TPM2 never enrolled).
func tpm2FollowsRequest(endpointManaged, persistentInstalledHost bool) bool {
	return endpointManaged || !persistentInstalledHost
}

// noSecureBootPolicyNotice tells the user why GjallarOS leaves Secure Boot
// alone on this machine. A generic "Secure Boot is disabled" read like a
// misdetection on the real HP, where nobody had said that ODDC simply has no
// Secure Boot setup for the model (2026-10-09).
func noSecureBootPolicyNotice(modelID string, firmwareOn, firmwareKnown bool) string {
	lines := []string{"Secure Boot setup:"}
	if modelID == "" {
		lines = append(lines, "  ODDC has no model for this machine, so it has no Secure Boot setup for it either.")
	} else {
		lines = append(lines, "  ODDC has no Secure Boot setup for "+modelID+" yet.")
	}
	lines = append(lines,
		"  GjallarOS will not install its own Secure Boot keys here, and TPM2 disk unlock stays off because it depends on them.",
	)
	switch {
	case !firmwareKnown:
		lines = append(lines, "  The firmware's Secure Boot state could not be read.")
	case firmwareOn:
		lines = append(lines,
			"  The firmware reports Secure Boot ON. The installed boot loader is not signed, so this",
			"  firmware will refuse to start it. Secure Boot has to be off in firmware setup before",
			"  the installed system boots.",
		)
	default:
		lines = append(lines, "  The firmware reports Secure Boot off; the machine boots as before.")
	}
	return strings.Join(lines, "\n") + "\n"
}
