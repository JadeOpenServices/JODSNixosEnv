
# System

The `system/` tree contains reusable NixOS configuration.

Main path:

    system/

This is where operating-system-level behavior belongs.

## Main blocks

### apps

    system/apps/

System-wide application services and application backends.

### hardware

    system/hardware/

Reusable hardware support and hardware policy.

Machine-generated hardware state belongs in `generated/hardware.nix`.
Canonical device-specific facts and exceptional policy belong in ODDC.

### management

    system/management/

Management integrations such as JODS.

Management features should remain separate from unrelated normal GjallarOS
features.

### security

    system/security/

System security configuration such as firewalling, SSH and other security
policy.

### tools

    system/tools/

System tools, commands and supporting utilities.

### users

    system/users/

System-level user and privilege configuration.

### virtualization

    system/virtualization/

Virtual-machine and virtualization support.

### wm

    system/wm/

System-side desktop and Wayland/window-manager enablement.

Per-user compositor and desktop configuration normally belongs under `user/wm/`.

### recovery

    system/recovery/

Recovery-image and installed recovery infrastructure.

See [Recovery](recovery.md).

## General rule

If a feature changes NixOS itself, system services, boot behavior, security,
hardware policy or machine-wide functionality, it normally belongs somewhere
under `system/`.

Generic composition imports these modules directly. Device-specific behavior
should enter through resolved ODDC capabilities or policy rather than through
parallel machine-specific module trees.
