package app

import "strings"

// tpm12Notice is printed when the only TPM is a TPM 1.2.
const tpm12Notice = `This machine has a TPM 1.2. GjallarOS cannot use Secure Boot or TPM disk unlock on it:
TPM 1.2 only supports SHA-1 measurements, and the boot policy needs TPM 2.0.
Some vendors ship a firmware update that turns it into a TPM 2.0; check the vendor's support page.
`

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
func noSecureBootPolicyNotice(modelID string) string {
	lines := []string{"Secure Boot setup:"}
	if modelID == "" {
		lines = append(lines, "  ODDC has no model for this machine, so it has no Secure Boot setup for it either.")
	} else {
		lines = append(lines, "  ODDC has no Secure Boot setup for "+modelID+" yet.")
	}
	lines = append(lines,
		"  GjallarOS will not install its own Secure Boot keys here, and TPM2 disk unlock stays off because it depends on them.",
	)
	return strings.Join(lines, "\n") + "\n"
}

// firmwareSecureBootNotice reports the firmware's own Secure Boot state when
// GjallarOS installs without its Secure Boot keys. It used to print only
// inside the no-ODDC-setup branch, which needs a TPM and secureBootPrompt, so
// other paths installed an unbootable system without a word. Firmware
// without a SecureBoot variable has no Secure Boot and boots anything.
func firmwareSecureBootNotice(firmwareOn bool, firmwareErr error) string {
	switch {
	case firmwareErr != nil:
		return "Firmware Secure Boot state could not be read (" + firmwareErr.Error() + ").\n"
	case firmwareOn:
		return strings.Join([]string{
			"Firmware Secure Boot is ON, but GjallarOS Secure Boot is not set up for this install.",
			"The installed boot loader is not signed, so this firmware will refuse to start it.",
			"Secure Boot has to be off in firmware setup before the installed system boots.",
		}, "\n") + "\n"
	}
	return "Firmware Secure Boot is off; the machine boots as before.\n"
}
