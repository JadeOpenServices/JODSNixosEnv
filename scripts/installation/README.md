# GjallarOS installer

Run this from the repository root on an existing NixOS installation:

```bash
./scripts/installation/install.sh
```

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
The same bootstrap installs `catppuccin-gtk` and sets
`environment.variables.GTK_THEME` to `Adwaita:dark`, a predictable GTK engine
for installer dialogs, while the
installer stylesheet supplies the Catppuccin palette and spacing. This avoids
theme-engine-specific rendering differences in Zenity.
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
installer, prerequisite bootstrapping and hardware regeneration are skipped.
This keeps an already-working machine-local setup intact while still allowing
settings changes and a normal rebuild.
LUKS and TPM2 prompts are skipped on normal reruns with the hardware step; use
`--refresh-hardware` to deliberately revisit them. Preset mode never suppresses
those security prompts when hardware configuration is being generated.

If a root-level `user.config.json` exists, the installer asks whether to reboot
automatically after a successful rebuild. That file is treated as local
machine configuration and is added to Git's local exclude list automatically.

Installer functions are kept as small, reusable Bash files in `functions/`.
Shared and installer-specific functions are organized beneath `functions/` for
readability, but their directory names are not part of the interface. The entry
point discovers and loads every `.sh` file recursively, so a category can be
renamed or reorganized without changing the installer. Only the location of the
`functions/` tree relative to `install.sh` matters, and the installer can be run
from any working directory.

The installed helper commands (`helpme`, `update`, `rebuild`, `cleanup`,
`thermal-status`, and `thermal-test`) are provided by `system/tools/` and use
the configured repository and hostname automatically.

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

Selecting `vscodium` as an editor installs VSCodium and the compatible
development extensions from nixpkgs. Its search and file-watcher exclusions
keep dependency/build trees out of indexing and AI context.

Microsoft Teams is provided as an Edge app with its own persistent profile.
HTTP(S) links to Teams open a small chooser: Microsoft Teams, the normal
browser, or cancel. Other links go straight to the configured browser.
