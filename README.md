# GjallarOS

GjallarOS is a modular NixOS workstation built around NixOS 26.05, Hyprland,
ODDC-backed hardware policy, generic system composition, and a reviewable installer.

## Quick start

```bash
git clone <your-repository-url> ~/.dotfiles
cd ~/.dotfiles
./scripts/installation/install.sh
```

For automated non-secret answers:

```bash
cp scripts/installation/user_PresetJSON/default.user.config.json user.config.json
```

Edit the copy, then run the installer. Secrets, LUKS passphrases, recovery
settings, generated hardware data, backups, and wallpapers are excluded from
Git.

## Included

- Hyprland-only desktop with Home Manager-managed user configuration.
- Noctalia shell styling with Stylix propagation for system applications.
- ODDC-backed device identity, capabilities, and hardware policy with generic system composition.
- Graphics, Wi-Fi, firmware, touchscreen, battery, and clamshell detection.
- VSCodium, Teams web app, system-wide Wine, and optional Nemu.
- Optional localhost-only Ollama AI with hardware-aware model selection.
- SOPS/age secrets, TPM2 LUKS enrollment, and recovery-key handling.
- Maintenance helpers for rebuilding, cleanup, thermal status, and updates.

## Configuration

`user.config.json` contains machine-local human intent. The installer normalizes
that intent into `generated/state.nix`; machine hardware lives in
`generated/hardware.nix`, and historical compatibility baselines live in
`generated/install-state.nix`. `deployment/release-policy.json` records the supported release. The
root flake, Home Manager, and Stylix follow 26.05; only the ComfyUI and
Tarantool development shells intentionally use unstable.

Hyprland bindings are maintained in
[`user/wm/hyprland/keybinds.json`](user/wm/hyprland/keybinds.json).

## Credits

GjallarOS builds on the ideas and groundwork of
[AlfheimOS](https://github.com/bakanura/JODSNixosEnv). Please send some love
their way as well.
