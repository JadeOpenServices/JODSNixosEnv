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

Both the shipped preset and the repository-local preset contain the complete
security controls:

```json
"recoveryEnable": false,
"jodsPrebootLockEnable": false,
"secureBootEnable": false,
"secureBootPrompt": true,
"endpointManagedDevice": false
```

Therefore the installer can be rerun directly. On an unmanaged Framework,
`secureBootPrompt=true` presents the Secure Boot choice. Accepting it prepares
keys and the signed system; a successful rebuild then offers a reboot directly
into firmware setup. The required firmware Setup Mode action, and possibly the
final Secure Boot enable toggle, remain physical firmware operations. Normal
boot resumes after exiting firmware, and the armed one-shot service performs
key enrollment automatically.
