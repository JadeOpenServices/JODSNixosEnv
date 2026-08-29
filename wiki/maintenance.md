# Maintenance

Validate before applying:

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
