
# GjallarOS architecture

GjallarOS is split into modular blocks.

The top-level repository structure tells you where a change belongs. Each block
has one main responsibility and may contain its own smaller modules.

This page is the overview. The linked pages explain each major block without
going into every individual implementation file.

## Configuration flow

    installer input
        |
        v
    gjallar-installer
        |
        v
    settings.nix
        |
        v
    flake.nix
        |
        +--> profile
        +--> system modules
        +--> Home Manager / user modules
        +--> packages
        +--> themes
        |
        v
    NixOS generation

## Main repository blocks

### Installer

The installer collects configuration, detects hardware, generates machine state
and coordinates deployment.

See [Installer](installer.md).

Main paths:

    scripts/installation/
    cmd/gjallar-installer/
    internal/installer/

### Profiles

Profiles connect a machine class to the reusable system and user modules.

See [Profiles](profiles.md).

Main path:

    profiles/

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

Hardware policy is reusable configuration under `system/hardware/`, while each
profile also has machine-generated hardware configuration.

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

It loads generated settings, selects the configured profile, imports NixOS and
Home Manager configuration, exposes repository packages and uses the configured
release policy.

### settings.nix

`settings.nix` is generated installer output.

It is the main configuration interface consumed by the Nix modules.

The normal flow is:

    user.config.json / prompts
        -> typed installer configuration
        -> normalization and detection
        -> settings.nix
        -> Nix modules

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
    machine profile             -> profiles/
    reusable hardware policy    -> system/hardware/
    package or wrapper          -> pkgs/
    theme                       -> themes/
    recovery                    -> system/recovery/ or scripts/recovery/

The goal is to keep each responsibility in its own module instead of growing
large central configuration files.
