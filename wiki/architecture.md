# Architecture

`flake.nix` imports the selected system profile and attaches Home Manager to
the same NixOS generation. SDDM waits for the primary Home Manager service.

- `system/` — NixOS services, hardware, security, virtualization, WM.
- `profiles/` — desktop, laptop, ThinkPad, Framework, and work profiles.
- `user/` — applications, editors, shells, and Hyprland configuration.
- `themes/` — palette and desktop theme definitions.
- `scripts/installation/` — installer and Bash functions.
- `deployment/` — release policy.
- `non-nix/` — runtime assets such as wallpapers and AGS sources.

`settings.nix`, generated hardware files, backups, and user presets are
machine-local and excluded from Git by the installer.
