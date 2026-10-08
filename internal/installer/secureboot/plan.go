package secureboot

import (
	"fmt"
	"slices"
	"strings"

	"github.com/JadeOpenServices/gjallarOS/internal/installer/oddc"
)

// OwnershipPlan explains, before anything is generated or changed, what the
// Secure Boot ownership transfer does to each firmware key database and what
// the user must keep to unlock and administer the machine afterwards. The
// per-database lines come from the same ODDC policy that enrollment enforces.
func OwnershipPlan(policy oddc.SecureBootFirmwarePolicy, luksTPM2 bool) string {
	firmwareName := strings.TrimSpace(policy.FirmwareName)
	if firmwareName == "" {
		firmwareName = "system firmware"
	}

	var b strings.Builder
	fmt.Fprintf(&b, `GJALLAROS SECURE BOOT - WHAT WILL CHANGE

Firmware: %s

`, firmwareName)

	for _, name := range []string{"PK", "KEK", "db", "dbx"} {
		switch {
		case slices.Contains(policy.Untouched, name):
			fmt.Fprintf(&b, "- %s: not changed.\n", name)
		case slices.Contains(policy.PreserveFirmwareBuiltin, name):
			fmt.Fprintf(&b, "- %s: the firmware's built-in manufacturer and Microsoft certificates stay trusted; GjallarOS adds its own key next to them.\n", name)
		case slices.Contains(policy.RequiredAbsent, name):
			fmt.Fprintf(&b, "- %s: you remove the manufacturer %s in firmware setup; GjallarOS enrolls its own %s in its place.\n", name, name, name)
		default:
			fmt.Fprintf(&b, "- %s: replaced by the GjallarOS key only.\n", name)
		}
	}

	b.WriteString(`
Afterwards only the GjallarOS Platform Key can change Secure Boot keys from
the running system. Boot loaders signed by GjallarOS or by a certificate kept
above still boot.

Firmware visits:
1. The installer reboots into firmware setup. Follow the device instructions
   shown before that reboot, save, and boot GjallarOS.
2. GjallarOS enrolls its keys and reboots into firmware setup once more.
   Enable Secure Boot, save, and boot GjallarOS.

What you need:
- access to firmware setup (know its password if one is already set)
- the Secure Boot recovery archive and passphrase shown next; store both
  offline, they restore the GjallarOS signing keys
`)
	if luksTPM2 {
		b.WriteString(`- your disk (LUKS) passphrase: the TPM opens the disk automatically only while
  Secure Boot and the signed boot chain are unchanged; otherwise it asks for
  the passphrase
`)
	}
	b.WriteString(`
Firmware setup itself stays open unless it has a supervisor password: anyone
at the machine could then turn Secure Boot off. GjallarOS can remind you to
set one during the second firmware visit. Use a new password, not the disk
passphrase, and store it offline; firmware setup, including Secure Boot, stays
locked without it.
`)
	return b.String()
}

// FirmwareLockSteps are the extra firmware-setup actions for the visit where
// Secure Boot gets enabled, when the user chose firmwarePasswordLock. The
// installer cannot set firmware passwords itself.
func FirmwareLockSteps(firmwareLock bool) []string {
	if !firmwareLock {
		return nil
	}
	return []string{
		"Set a firmware supervisor (administrator) password. Use a new password, not the disk passphrase.",
		"Store that password offline. Changing firmware settings, including Secure Boot, requires it from now on.",
	}
}
