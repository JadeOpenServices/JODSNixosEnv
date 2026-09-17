
# User and Home Manager

The `user/` tree contains per-user configuration managed through Home Manager.

Main path:

    user/

This is where the interactive desktop environment and user applications are
configured.

## Main blocks

### apps

    user/apps/

User applications, wrappers and desktop integration.

### browsers

    user/browsers/

User-visible browser configuration.

### editors

    user/editors/

Editor configuration such as VSCodium and Neovim.

### services

    user/services/

Per-user background owners such as storage maintenance and application
resource QoS. `resource-qos.nix` owns the app2unit launch boundary and workload
family policy.

### shells

    user/shells/

Shells, prompts and shell tooling.

### wm

    user/wm/

Per-user window-manager, compositor, shell-panel and desktop configuration.

This includes areas such as Hyprland configuration and user shell integration.

## System versus user

A useful split is:

    system/
        machine-wide enablement and services

    user/
        per-user configuration and desktop behavior

For example, system-level Wayland/session support belongs under `system/wm/`,
while user keybinds and compositor settings belong under `user/wm/`.
