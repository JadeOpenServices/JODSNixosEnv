
# Installer

The GjallarOS installer is a typed Go application with a small shell launcher.

Its job is to turn installer input and detected machine state into a valid
GjallarOS configuration and coordinate installation.

## Main paths

    scripts/installation/
    cmd/gjallar-installer/
    internal/installer/

## Launcher

The normal entry point is:

    scripts/installation/install.sh

The script builds the current repository's installer package and starts:

    gjallar-installer

The actual installer behavior lives in Go.

## Command entry point

    cmd/gjallar-installer/

This is the executable entry point.

It hands control to the installer application under `internal/installer/`.

## Installer modules

The main installer implementation lives under:

    internal/installer/

Major blocks include:

    app/
    bootstrap/
    config/
    deploy/
    discovery/
    diskcrypto/
    firmware/
    hardwareconfig/
    nixrender/
    policy/
    prompt/
    release/
    secrets/
    secureboot/

These are separate modules so prompting, configuration, rendering, hardware,
security and deployment do not all live in one installer file.

## Application

    internal/installer/app/

The application coordinates the overall installer transaction.

At a high level it:

1. resolves the repository
2. validates the environment
3. discovers hardware
4. reads installer configuration
5. collects interactive choices
6. normalizes configuration
7. generates machine-specific state
8. renders `generated/state.nix`
9. validates the resulting configuration
10. deploys the configured NixOS generation
11. handles later installer lifecycle stages

## Configuration

    internal/installer/config/

This is the typed configuration contract used by the installer.

Installer input should enter the rest of the installer through this typed
representation rather than being read independently by desktop or system
modules.

## Discovery

    internal/installer/discovery/

Discovery inspects the current machine and provides hardware/environment facts
to the installer.

Detected information contributes machine facts and generated configuration.
Canonical device identity and exceptional hardware policy are resolved through
ODDC.

## Rendering

    internal/installer/nixrender/

Rendering converts typed installer values into safe Nix values.

The main generated machine-local configuration is:

    generated/state.nix

## Hardware configuration

    internal/installer/hardwareconfig/

Hardware configuration generation uses NixOS hardware discovery and stores the
machine-local result separately from reusable repository configuration.

Canonical target:

    generated/hardware.nix

## Deployment

    internal/installer/deploy/

The current normal deployment path validates the generated NixOS configuration
and prepares the next boot generation.

Conceptually:

    nixos-rebuild dry-build
        ->
    nixos-rebuild boot
        ->
    reboot

## Disk security

    internal/installer/diskcrypto/
    internal/installer/secureboot/
    internal/installer/firmware/

Disk encryption and Secure Boot are kept separate from ordinary configuration
rendering because they operate across security and firmware boundaries.

## Installer validation

Repository-level installer checks are implemented separately under:

    internal/installercheck/

The standard repository check is:

    gjallarctl check --repo .

Installer changes should also run the focused tests and the full Go test suite.
