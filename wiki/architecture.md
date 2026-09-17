
# GjallarOS architecture

GjallarOS is split into modular blocks.

The top-level repository structure tells you where a change belongs. Each block
has one main responsibility and may contain its own smaller modules.

This page is the overview. The linked pages explain each major block without
going into every individual implementation file.

## Configuration flow

    user.config.json / installer prompts
        |
        v
    gjallar-installer
        |
        +--> generated/state.nix
        +--> generated/hardware.nix
        +--> generated/install-state.nix
        |
        v
    flake.nix
        |
        +--> generic system modules
        +--> Home Manager / user modules
        +--> ODDC-resolved hardware policy
        +--> packages
        +--> themes
        |
        v
    NixOS generation

## Ownership and routing

GjallarOS keeps hardware facts, human intent and operating-system policy
separate. ODDC owns canonical device identity, capabilities and validated
device-specific facts. `user.config.json` owns direct machine-local user
intent. Generic NixOS and Home Manager modules own reusable operating-system
policy.

State follows one routed path instead of being rediscovered by each consumer:

    hardware facts -> ODDC / generated hardware state
    user intent    -> typed installer config -> generated/state.nix
    OS policy      -> one generic module owner -> consumers

`rebuild` refreshes direct user-owned generated-state assignments through the
same typed renderer used by installation. Hardware, AI discovery and other
derived facts remain installer-owned. `gjallar-preflight` performs cheap static
wiring checks before Nix evaluation.

Application resource management follows the same model. `gjallar-run` uses
app2unit workload identities for desktop, JODS, AI and explicit background
work. Identity and policy remain separate; only normal desktop scopes receive
Hyprland focus-based CPU-weight changes.

## Main repository blocks

### Installer

The installer collects configuration, detects hardware, generates machine state
and coordinates deployment.

See [Installer](installer.md).

Main paths:

    scripts/installation/
    cmd/gjallar-installer/
    internal/installer/

### ODDC

ODDC is the canonical device identity, hardware composition, capability,
validated quirk, and exceptional hardware-policy authority.

Generic GjallarOS modules consume resolved ODDC data without selecting a
machine-specific system profile.

Main path:

    oddc/

### System

System modules configure NixOS itself.

See [System](system.md).

Main path:

    system/

### User

User modules configure the interactive desktop and Home Manager environment.

See [User and Home Manager](user.md).

Main path:

    user/

### Packages

Repository-owned packages, wrappers and overlays live in the package layer.

See [Packages](packages.md).

Main paths:

    pkgs/
    lib/

### Hardware

Generic hardware behavior lives under `system/hardware/`. Canonical device
identity and exceptional device policy live in ODDC. Machine-local generated
hardware facts live in `generated/hardware.nix`.

See [Hardware](hardware.md).

### Recovery

Recovery provides the bootable maintenance and installer environment and the
installed recovery lifecycle.

See [Recovery](recovery.md).

Main paths:

    system/recovery/
    scripts/recovery/

## Root files

### flake.nix

`flake.nix` ties the repository together.

It loads generated machine-local state, imports the generic NixOS and Home
Manager composition, exposes repository packages, integrates resolved ODDC
policy, and uses the configured release policy.

### Generated machine state

The installer keeps derived machine-local state separate from reusable
repository configuration.

The normal flow is:

    user.config.json / prompts
        -> typed installer configuration
        -> normalization
        -> generated/state.nix
        -> generic Nix modules

Hardware discovery is written separately to:

    generated/hardware.nix

Historical NixOS and Home Manager compatibility baselines live in:

    generated/install-state.nix

### user.config.json

`user.config.json` is machine-local installer input.

It should not be treated as reusable repository configuration and should not be
committed.

The default installer preset lives at:

    scripts/installation/user_PresetJSON/default.user.config.json

## Where changes normally belong

    installer behavior          -> internal/installer/
    installer command           -> cmd/
    system configuration        -> system/
    user application/config     -> user/
    generic hardware behavior  -> system/hardware/
    device identity/policy       -> oddc/
    machine-local hardware       -> generated/hardware.nix
    package or wrapper          -> pkgs/
    theme                       -> themes/
    recovery                    -> system/recovery/ or scripts/recovery/

The goal is to keep each responsibility in its own module instead of growing
large central configuration files.
