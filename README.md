# GjallarOS

![NixOS](https://img.shields.io/badge/NixOS-26.05-5277C3?logo=nixos&logoColor=white)

GjallarOS is a modular, hardware-aware NixOS workstation for people who want
a beautiful desktop without turning maintenance into a second job. It brings
the system, user environment, hardware tuning, and installer into one
reviewable flake.

## What it provides

- Hyprland, Sway, KDE, GNOME, and Wayfire building blocks.
- Automatic graphics, Wi-Fi, firmware, touchscreen, and laptop detection.
- Framework 12/13/16 profiles with safe fan and battery controls.
- Suspend-friendly clamshell behavior, power-button protection, and optional
  TPM2-backed LUKS unlock with a printed recovery key.
- System-wide Wine WoW64 staging with Win11 prefix tooling, DXVK, VKD3D, and
  NTFS support; Steam and Proton remain available for games.
- VSCodium, Teams as an Edge app, SOPS secrets, fingerprint enrollment, and
  focused maintenance tools.
- Optional Nemu virtualization with explicitly confirmed GPU passthrough.

The AI path is local-first: Ollama is bound to localhost, limited to one
loaded model/request, and tuned for flash attention with a memory-conscious
KV cache. Internet research is deliberately not implicit; an explicit,
approved retrieval tool can be added later without giving the model blanket
network access.

The local-agent policy in [`AGENTS.md`](./AGENTS.md) requires checkpointing and
task splitting when the configured context budget or available VRAM becomes
constrained. This keeps 8 GiB GPUs responsive instead of allowing one huge
request to exhaust memory.

Use `gjallar-research <https-url>` when you explicitly want reference
material. It allows only HTTPS and a small documentation allowlist, limits
responses to 2 MiB/15 seconds, and prints the result for review; it is not an
automatic browsing capability. The policy is recorded in
`system/apps/ai/retrieval-policy.json` for a future approved endpoint.

The default Hyprland bindings are maintained in
[`user/wm/hyprland/keybinds.json`](./user/wm/hyprland/keybinds.json), so they
can be changed without hunting through Nix modules.

## Install

This project targets NixOS 26.05 and keeps its nixpkgs input locked. On an
existing NixOS installation:

```bash
git clone <your-repository-url> ~/.dotfiles
cd ~/.dotfiles
./scripts/installation/install.sh
```

The installer verifies the release, offers to bootstrap missing tools, asks
for user, profile, hardware, Framework, work-account, scrobbling, Nemu, TPM2,
and LUKS choices, then shows a summary before writing files or rebuilding.
Hardware configuration is backed up before replacement. Use `--no-rebuild`
to review the generated configuration first.

After the system switch, apply the user environment with:

```bash
home-manager switch --flake .
```

## Release policy

`deployment/release-policy.json` records the supported release and provides a
stable place for a future endpoint to select tested release profiles for user
groups. Updates should refresh and review `flake.lock`, then promote the
tested commit.

## Screenshots

Add desktop screenshots here as the setup evolves.

## Credits

GjallarOS builds on the ideas and groundwork of
[AlfheimOS](https://github.com/bakanura/JODSNixosEnv). If this project helps
you, please send some love their way as well.
