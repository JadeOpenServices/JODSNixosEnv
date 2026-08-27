# AlfheimOS installer

Run this from the repository root on an existing NixOS installation:

```bash
./scripts/installation/install.sh
```

The installer targets NixOS 26.05 and stops before making changes if another
release is detected. The release/channel policy is kept in
`deployment/release-policy.json` for future centrally managed updates.

The wizard asks for every setting in `settings.nix`, discovers available
profiles and modules, writes your selections, and can generate hardware
configuration before running `nixos-rebuild`.

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
- `--help` — show usage.

Installer functions are kept as small, reusable Bash files in `functions/`.

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
selects AMD, Intel, or NVIDIA graphics settings, enables firmware, and adds the
matching common Wi-Fi kernel driver when it can identify one. Unknown hardware
keeps the safe generic firmware and NetworkManager setup.

Selecting `vscodium` as an editor installs VSCodium and the compatible
development extensions from nixpkgs. Its search and file-watcher exclusions
keep dependency/build trees out of indexing and AI context.

Microsoft Teams is provided as an Edge app with its own persistent profile.
HTTP(S) links to Teams open a small chooser: Microsoft Teams, the normal
browser, or cancel. Other links go straight to the configured browser.
