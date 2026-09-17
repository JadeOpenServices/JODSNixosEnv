# Maintenance

Run the fast repository preflight before expensive Nix work:

```bash
gjallar-preflight
```

`rebuild` runs the same preflight automatically. It validates routed machine
state, required owners, flake-input wiring and likely orphan modules before
`nixos-rebuild` starts. Warnings identify cleanup candidates without blocking
the rebuild; failed structural checks stop before privilege escalation.

For a full Nix-level dry run:

```bash
nix build .#nixosConfigurations.gjallarOS.config.system.build.toplevel --dry-run
```

The installer uses `nixos-rebuild boot`; reboot to activate it. Home Manager
backs up conflicting files with `.hm-bak`. Failed rebuilds leave the previous
generation active.

Useful checks:

```bash
systemctl --failed
systemctl status home-manager-<username>.service
readlink -f ~/.config/hypr/hyprland.conf
```
