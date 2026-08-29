# Installation

Run from an existing NixOS system:

```bash
./scripts/installation/install.sh
```

The installer verifies NixOS 26.05, bootstraps missing tools, optionally runs
fwupd, detects hardware, writes settings, validates the flake, and installs
the next generation with `nixos-rebuild boot`. Reboot afterward; the current
session is not restarted during installation.

Preset setup:

```bash
cp scripts/installation/user_PresetJSON/default.user.config.json user.config.json
```

Presets answer normal settings but never bypass LUKS/TPM2 or secret prompts.
Use `--no-rebuild`, `--skip-hardware`, or `--refresh-hardware` as needed.
Reruns preserve existing machine-local setup.
