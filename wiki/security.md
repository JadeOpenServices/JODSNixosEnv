# Security

SOPS/age protects service secrets. Last.fm and ListenBrainz credentials are
entered in the terminal and encrypted; never put tokens in JSON.

The LUKS flow supports existing-key retention, TPM2 unlock enrollment,
confirmed key rotation, and a separate displayed recovery key. Recovery keys
are not written to repository files.

The optional work account is `<username>-corp` with a separate home directory
and reduced application set.

## Secure Boot and measured boot

Set `secureBootPrompt=true` to let the installer offer GjallarOS Secure Boot
on an unmanaged supported device.

Secure Boot firmware behavior is resolved from the detected ODDC device layer.
ODDC contains typed declarative policy only: it may describe the firmware name,
Setup Mode strategy, required EFI variables, firmware trust material that must
be preserved, variables that must remain untouched, and user-facing firmware
instructions. ODDC cannot provide commands, scripts, executables, URLs, hooks,
or arbitrary shell input.

The installer creates per-device GjallarOS keys, writes an encrypted recovery
archive, requires confirmation that the recovery material was saved, installs a
signed Lanzaboote generation, verifies the signed EFI artifacts, records
GjallarOS ownership state, and snapshots the resolved firmware policy for the
transaction.

Firmware actions remain physical operations. The installer displays the
instructions from the validated ODDC policy for the detected device. GjallarOS
then independently verifies the resulting UEFI state before enrollment. Unknown
or unsupported firmware states fail closed.

During enrollment, trusted GjallarOS code controls all privileged mutation.
The firmware policy may require selected firmware-provided KEK/db material to
be retained while GjallarOS establishes its own PK/KEK/db ownership. dbx or
other variables marked untouched by policy are not modified.

After enrollment, GjallarOS verifies:

- Secure Boot ownership matches the locally provisioned GjallarOS keys;
- the expected PK/KEK/db certificates are active;
- signed boot artifacts verify successfully;
- Setup Mode has been left;
- Secure Boot enforcement is enabled before installation completion.

The transaction survives required firmware reboots through root-only state
markers. `gjallar-secure-boot-enroll.service` performs policy-constrained
enrollment, `gjallar-secure-boot-finalize.service` verifies the final Secure
Boot state, and `gjallar-installer-post-secure-boot.service` completes the
installer transaction only after those checks pass.

When TPM2-backed LUKS unlock is enabled, measured-boot enrollment runs after
Secure Boot ownership has been finalized. It requires the PCR-lock policy,
verifies the human recovery passphrase before TPM enrollment, refuses ambiguous
pre-existing TPM2 tokens, and preserves the human recovery path.

If no TPM2 device is detected while TPM-dependent security is requested, the
interactive installer explains that GjallarOS cannot use its Secure Boot /
measured-boot policy on that machine and asks whether to continue with those
features disabled. Accepting the downgrade updates a loaded `user.config.json`
so future runs do not request those unavailable features. Unattended installs
never silently weaken the requested security configuration.

Check Secure Boot state with:

```bash
gjallar-secure-boot status
systemctl status gjallar-secure-boot-enroll.service
systemctl status gjallar-secure-boot-finalize.service
```

The `gjallar-secure-boot enroll` wrapper is available for the trusted
enrollment transaction. Firmware-device information can be inspected with
`gjallar-secure-boot firmware`.

`endpointManagedDevice=true` suppresses interactive Secure Boot enrollment
and recovery-material prompts. JODS must provision and escrow unique per-device
material according to its managed-device policy; fleet-wide shared private
signing keys are forbidden.

## Fingerprint enrollment

On an unmanaged device, the first fingerprint setup uses the account password.
After enrollment metadata exists, every later add or delete requires a fresh
administrator/root-password polkit authorization. Fingerprints cannot authorize
that prompt, and authorization is not cached.

On a JODS-managed device, local enrollment and deletion default to denied. The
signed JODS policy may set `jodsFingerprintEnrollmentAllowed=true`; this permits
the physical enrollment UI but retains the fresh administrator-password gate.
Raw fingerprint templates remain local under fprintd and are never sent to JODS.


### Lanzaboote external kernel

Do not individually sign `/boot/EFI/nixos/kernel-*.efi`.

GjallarOS/Lanzaboote keeps this external kernel byte-identical to the Nix-store
kernel. The signed generation stub verifies its hash. Adding an Authenticode
signature changes the kernel bytes and causes `Kernel hash does not match` /
`SECURITY_VIOLATION`.
