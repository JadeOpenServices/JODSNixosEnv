# GjallarOS installer

Run this from the repository root on an existing NixOS installation:

```bash
./scripts/installation/install.sh
```

The short bootstrapper requires only Nix. It builds and starts
`gjallar-installer` without changing the current shell environment. After one
approval, the Go preflight persistently adds Go, PCI/firmware tools, Git,
HTTPS/GTK helpers, SOPS/Age, password hashing, and disk-encryption tooling to
`/etc/nixos/configuration.nix`, rebuilds, then resumes the installation.

## Preset automation

Copy the preset with:

```bash
cp scripts/installation/user_PresetJSON/default.user.config.json user.config.json
```

Then edit `user.config.json` and run the installer.
The preset answers the non-secret configuration automatically. Hardware
profiling remains dynamic, GPU passthrough IDs are detected automatically, and
Last.fm/ListenBrainz usernames and tokens are always requested in the terminal;
never place those credentials in this file. Remove or rename the file to return
to fully interactive mode.

The installer estimates the weather location from the current public network
address and asks the user to confirm it. If rejected or unavailable, it asks
for city and country explicitly. The detected value is never accepted silently.

In the preset, `username` is the Linux account and the single identity input.
`dotfilesDir` is derived automatically as `/home/<username>/Documents/gjallarOS`.
`workUserEnable` controls the optional work account, whose default name is
`<username>-corp`. The `name` field is the human-readable Git commit author
name; it is not the Linux username or GitHub username. `githubUsername` is a
separate optional account label.

Optional `backgroundNormal` and `backgroundWork` values select different
wallpapers for the normal and work accounts. Use an absolute path, a path
inside the repository (for example `non-nix/wallpapers/nord.png`), or an
HTTPS URL. HTTPS backgrounds are downloaded into `non-nix/wallpapers/` using a
stable URL-derived filename; an empty value uses the selected theme wallpaper.
`backgroundGaming` is reserved for a future gaming account and is intentionally
unused for now. Downloaded `user-*` wallpapers are added to Git's local exclude
file, just like `settings.nix`, generated hardware backups, and
`user.config.json`.
Set `runUpdateChecks` to `true` or `false` in preset mode to automatically run
or skip the fwupd update check without being prompted.
Set `keyboardLayout` to an XKB layout such as `us` or `de`. The installer
normalizes `de` and `de-latin1` to the valid standard German XKB layout `de`
for both Hyprland and SDDM.
Set `debugFunctions` to `true` to enable the boot diagnostic service. It logs
the display manager, NetworkManager, Home Manager, failed units, and
Hyprland-config state. It also records the actual Hyprland-session environment,
binary paths, running processes, monitors, and clients after login. Read
`/var/lib/gjallar-diagnostics/boot-*.log` and
`~/.local/state/gjallar-diagnostics/hyprland-*.log`.
The Hyprland report includes `hyprctl configerrors`, which exposes invalid
configuration lines and failed commands. The report is a systemd user service
that remains active for the logged-in Hyprland session.

The installer targets NixOS 26.05 and stops before making changes if another
release is detected. The release/channel policy is kept in
`deployment/release-policy.json` for future centrally managed updates. The root
flake and stable shell flakes track its `nixpkgsInput`; only the explicitly
listed development shells (currently ComfyUI and Tarantool) intentionally use
`unstableNixpkgsInput` for fast-moving GPU/build tooling. Flake input URLs must
remain literal by Nix design, so the installer validates the running release
against this policy before proceeding.

Every run offers an explicit firmware-update check. If accepted, it refreshes
fwupd metadata and applies available updates before continuing; declining it or
an unavailable update is non-fatal.
The bootstrap step enables `services.fwupd` in `/etc/nixos/configuration.nix`
as well as installing the command, so firmware updates work before the first
GjallarOS rebuild when possible.

The wizard asks for every setting in `settings.nix`, discovers available
profiles and modules, writes your selections, and can generate hardware
configuration before running `nixos-rebuild`.
`settings.nix` is rendered and atomically replaced by `gjallarctl`; all string
values escape Nix interpolation markers as well as quotes and backslashes.
Machine-local Git protection is also owned by `gjallarctl`, with repository
containment checks and fixed-argument Git invocations.
Hardware generation uses a Go plan/apply action with a validated profile target,
timestamped backup, fixed `nixos-generate-config` arguments, and atomic replacement.
Deployment uses a Go plan/apply action: `nixos-rebuild dry-build` must pass before
the next boot generation is installed, and both privileged commands use fixed arguments.
Because Nix flakes use the Git snapshot, newly added profile metadata is
automatically staged locally before rebuilding; no commit or remote push is
performed.
The installer detects whether it is running on a laptop or desktop before
showing profiles: desktops are limited to the standard `desktop` profile,
while laptops receive laptop-oriented profiles (including `thinkpad` and any
future `framework*` profiles added to the repository). Presets may explicitly
choose a profile and are not overridden by this filter.

When run inside a graphical session, the installer uses GTK dialogs through
Zenity. Zenity is bootstrapped into the NixOS configuration automatically;
TTY, SSH, live-media, and headless runs continue using the terminal interface.
On an existing install, a missing Zenity package also triggers the small
prerequisite bootstrap so the next installer prompts can use GTK.
The same bootstrap sets `environment.variables.GTK_THEME` to `Adwaita:dark`,
a predictable GTK engine for installer dialogs. The active Noctalia/Stylix
palette owns normal desktop theming.
Closing or pressing Escape in a GTK prompt opens a quit confirmation; choosing
No returns to the interrupted prompt.

Local AI is optional. If disabled, Ollama, OpenCode, the AI wrapper commands,
and AI profiling are all omitted from the generated system.

Before rebuilding, make sure your own SOPS age key exists and that
`secrets/default.yaml` is encrypted for it. The installer checks this and
prints a warning when the secrets cannot be decrypted; it does not create or
guess service credentials.

If you enable scrobbling, it asks for Last.fm and/or ListenBrainz usernames and
tokens, then encrypts them with your local age key. If the generated hardware
configuration contains LUKS, it verifies the current key and offers an
explicitly confirmed key rotation. Declining any confirmation leaves the
existing key unchanged.

When a TPM2 device is present, the installer can enroll a TPM2 LUKS unlock
slot. The original passphrase is retained as recovery; firmware or boot-chain
changes can still require it.

The LUKS step also displays a separate 64-character recovery key. It is
enrolled in its own key slot whether or not the normal key is rotated, and the
installer requires confirmation that it was saved.

Options:

- `--no-rebuild` — write `settings.nix` without applying it.
- `--skip-hardware` — do not generate a hardware configuration.
- `--refresh-hardware` — on an existing GjallarOS install, deliberately
  regenerate the selected profile's hardware configuration.
- `--help` — show usage.

When rerun from a checkout whose `settings.nix` was generated by this
installer, prerequisite `/etc/nixos/configuration.nix` bootstrapping and
hardware regeneration are skipped. A rerun never switches back to the base
configuration before deploying the selected GjallarOS flake.
This keeps an already-working machine-local setup intact while still allowing
settings changes and a normal rebuild.
LUKS and TPM2 prompts are skipped on normal reruns with the hardware step; use
`--refresh-hardware` to deliberately revisit them. Preset mode never suppresses
those security prompts when hardware configuration is being generated.

If a root-level `user.config.json` exists, the installer asks whether to reboot
automatically after a successful rebuild. That file is treated as local
machine configuration and is added to Git's local exclude list automatically.

`gjallar-installer` owns discovery, terminal/GTK prompts, policy, rendering, and
action sequencing. `install.sh` builds that binary without changing the current
shell environment. The Go preflight proposes persistent prerequisite additions
to `/etc/nixos/configuration.nix`, applies them only after approval, rebuilds,
then continues automatically. The former sourced Bash implementation has been
removed.

The installed helper commands (`helpme`, `update`, `rebuild`, `cleanup`,
`thermal-status`, and `thermal-test`) are provided by `system/tools/` and use
the configured repository and hostname automatically.

## Recovery and JODS preboot policy

`recoveryEnable` is disabled by default. When enabled, the bootloader exposes a
console-only, Secure-Boot-signed `gjallar-recovery` specialisation. A separate
minimal recovery/installer ISO is available as
`nix build .#gjallar-recovery-iso`. It has SSH disabled, a default-deny
firewall, and the tools required to unlock, repair, or install GjallarOS.

JODS fresh-install media should reserve one dedicated exactly 12 GiB partition and
install a signed recovery release with `scripts/recovery/install-partition.sh`.
If a GPT disk already has sufficient unallocated space,
`scripts/recovery/create-partition.sh DISK` can create only that partition.
It refuses to shrink filesystems and requires a separate exact confirmation.
The helper refuses whole disks, mounted partitions, undersized/oversized
targets, and any image whose pinned Ed25519 signature, size, or SHA-256 digest
does not match. It creates the FAT32 label `JODSRECOV`, extracts the ISO, and
signs and verifies every EFI executable with the endpoint's installed Secure
Boot key. It requires the exact destructive confirmation phrase. Existing
enrolled machines are never repartitioned automatically. Their recovery
partition is added only through an explicitly approved maintenance operation.

Release process:

```text
nix build .#gjallar-recovery-iso
scripts/recovery/sign-image.sh result/iso/*.iso OFFLINE-ED25519-KEY release/
scripts/recovery/verify-image.sh IMAGE MANIFEST MANIFEST.sig PINNED-PUBLIC.pem
sudo scripts/recovery/install-partition.sh PARTITION IMAGE MANIFEST MANIFEST.sig PINNED-PUBLIC.pem
```

The interactive installer performs this transaction after a successful NixOS
deployment when recovery is enabled and the operator opts in. It reuses a
detected `JODS-RECOVERY` partition or offers creation from unallocated GPT
space. For unattended installs, pass `--recovery-partition` or
`--recovery-disk` together with the runtime-only `--recovery-signing-key`.
These paths are never rendered into `settings.nix`.

## Optional JODS enrollment

The installer asks whether JODS should manage the machine. Managed installs
pin the JODS executor from `flake.lock`, validate an HTTPS endpoint and the
64-hex-character Ed25519 policy key, and render only public configuration.
Insecure TLS is limited to an explicitly confirmed local-development flow.

Enabling the module does not contact JODS. After a successful rebuild the
installer writes `/var/lib/gjallarOS/installation-complete`, starts
`jods-mdm-agent-enroll.service` once, then enables its retry timer. Failed or
deferred rebuilds remain `configured, not contacted`. Device identity stays in
`/var/lib/jods-mdm-agent` and is preserved by ordinary rebuilds and in-place
installer runs.

Keep the release signing key offline. JODS stores the public key and signed
artifacts, never the private release key. The bootable recovery specialisation
is produced and signed by Lanzaboote using the endpoint's per-device Secure
Boot keys; the FAT32 partition contains the independently bootable recovery
environment used by that trusted entry.

`jodsPrebootLockEnable` is also disabled. It publishes declarative JODS policy
intent at `/etc/jods/preboot-policy`, but does not claim to be an anti-theft
lock. That requires a later Secure Boot, measured-boot, remote-attestation, and
revocable LUKS-key-release design; a local boot flag alone is bypassable.

Framework profiles also expose `secureBootEnable`, disabled by default. The
Framework provisioning sequence is:

```text
gjallar-secure-boot create-keys
set secureBootEnable=true and rebuild
verify with gjallar-secure-boot status
installer arms enrollment and opens Framework firmware setup
delete ONLY PK, keep KEK, DB, and DBX intact, then boot with Secure Boot disabled
DO NOT use "Erase All Secure Boot Settings"; that would also remove KEK, DB, and DBX
boot service enrolls per-device keys and verifies signed artifacts
reboot and verify again
```

The Linux-only enrollment command does not add Microsoft keys. It retains the
Framework firmware-builtin keys needed by platform devices and firmware update
flows. Keep `/var/lib/sbctl` private and backed up; JODS should escrow recovery
material without distributing a shared fleet-wide private signing key.

`secureBootPrompt=true` asks unmanaged Framework users whether to prepare
Secure Boot, including when a preset is loaded. `endpointManagedDevice=true` suppresses
that prompt: JODS must provision and escrow per-device keys itself. On an
unmanaged device, the installer creates an encrypted recovery archive under
`/var/lib/gjallarOS/recovery`, displays its generated passphrase outside shell
history before rebuild or reboot, and requires two save confirmations.
The whole installer transaction remains armed across both firmware visits.
After `gjallar-secure-boot-finalize.service` proves that Secure Boot is
enforcing and that GjallarOS owns the expected PK/KEK/db hierarchy,
`gjallar-installer-post-secure-boot.service` performs the final installer
verification and only then records installation completion.

The reboot-time continuation is deliberately user-visible. Progress and
failures are written to `/var/lib/gjallarOS/installer-status.txt` and the
system journal, are broadcast to logged-in terminals, and use a GTK/Zenity
dialog when a graphical user session is available. If no GUI or terminal is
currently attached, the persistent status file and journal retain the result.
Failures leave the installer continuation marker in place so a reboot or
repair cannot be mistaken for a completed installation.

After a successful installer rebuild, a root-only marker arms automatic
enrollment. The installer can reboot directly into firmware settings. Framework Setup Mode requires clearing ONLY PK while keeping KEK, DB, and DBX intact; once Linux boots in Setup
Mode, `gjallar-secure-boot-enroll.service` enrolls the keys with
`--firmware-builtin=db,KEK`, verifies the signed artifacts, and deletes its
marker. Rerunning the installer detects keys already enrolled by an interrupted
service, clears the stale enrollment marker, and resumes at the final firmware
enable/verification step without enrolling keys again.

Laptop profiles include `thermald`, `auto-cpufreq`, UPower, conservative
battery charge thresholds, and dock-friendly lid behavior. Fan curves are not
forced because the reference controller only supports specific hardware.

The installer asks whether the machine is a Framework laptop and offers
profiles for Framework 13, 16, and 12. Framework fan control and charge
thresholds are enabled only after that choice; each model’s JSON profile lives
in `system/hardware/framework/profiles/` for future tuning.

The installer also detects the graphics and Wi-Fi hardware with `lspci`. It
selects AMD, Intel, or NVIDIA graphics settings and enables firmware. Wi-Fi
drivers are left to kernel autodetection to avoid forcing a mismatched module;
unknown hardware keeps the safe generic firmware and NetworkManager setup.

`dockerEnable=true` enables Docker-compatible commands through rootless Podman
and `podman-compose`. GjallarOS never starts the root Docker daemon and never
adds desktop users to the root-equivalent `docker` group.

The rootless Podman user socket is available on demand for explicit local use,
but GjallarOS does not export it globally or keep it alive through user
lingering. Services must not mount the socket merely to collect status or logs.
Compose files should publish host ports `>= 1024`; GjallarOS does not weaken
`net.ipv4.ip_unprivileged_port_start` just to make rootless containers claim
ports 80 or 443. Put a system reverse proxy in front when standard public ports
are required.

OCI short names resolve only against `docker.io`, preventing Podman from asking
which registry to use. Repository-owned Dockerfiles and Compose files should
still use fully qualified references such as `docker.io/library/alpine:3.22`.

Selecting `vscodium` as an editor installs VSCodium and the compatible
development extensions from nixpkgs. Its search and file-watcher exclusions
keep dependency/build trees out of indexing and AI context.

Microsoft Teams is provided as an Edge app with its own persistent profile.
HTTP(S) links to Teams open a small chooser: Microsoft Teams, the normal
browser, or cancel. Other links go straight to the configured browser.


Secure Boot note: the Lanzaboote external `/boot/EFI/nixos/kernel-*.efi` must
not be individually signed. It must remain byte-identical to the Nix-store
kernel because the signed generation stub verifies its hash.
