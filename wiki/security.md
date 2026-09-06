# Security

SOPS/age protects service secrets. Last.fm and ListenBrainz credentials are
entered in the terminal and encrypted; never put tokens in JSON.

The LUKS flow supports existing-key retention, TPM2 unlock enrollment,
confirmed key rotation, and a separate displayed recovery key. Recovery keys
are not written to repository files.

The optional work account is `<username>-corp` with a separate home directory
and reduced application set.

## Framework Secure Boot

Set `secureBootPrompt=true` to let the installer offer Secure Boot on an
unmanaged Framework laptop. The installer creates per-device keys, writes an
encrypted recovery archive, requires two confirmations that its passphrase was
saved, installs a signed Lanzaboote generation, verifies the signed EFI
artifacts, and arms boot-time enrollment.

The installer can reboot directly into Framework firmware settings. Firmware
still requires one physical action: delete ONLY PK to enter Setup Mode. Keep KEK, DB, and DBX intact. Do not select **Erase All Secure Boot Settings**. Do not clear KEK, DB, or DBX. Exit with Secure Boot disabled and boot GjallarOS normally.

On that boot, `gjallar-secure-boot-enroll.service` detects Setup Mode, enrolls
the per-device keys with `sbctl enroll-keys --firmware-builtin`, verifies the
signed artifacts, and removes its root-only enrollment marker. If Framework
firmware does not enable Secure Boot automatically after enrollment, enter the
firmware menu once more and enable it. Firmware policy prevents the operating
system from safely automating those menu actions.

Check the result with:

```bash
gjallar-secure-boot status
systemctl status gjallar-secure-boot-enroll.service
```

`endpointManagedDevice=true` suppresses all interactive enrollment and recovery
material prompts. JODS must provision and escrow unique per-device keys before
managed enrollment is enabled; fleet-wide shared private keys are forbidden.

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
