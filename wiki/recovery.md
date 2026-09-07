
# Recovery

GjallarOS recovery provides a bootable maintenance and installer environment
plus installed-system recovery support.

Main paths:

    system/recovery/
    scripts/recovery/

## Recovery image

    system/recovery/image.nix

This defines the minimal recovery/installer environment using NixOS installer
image modules.

The recovery image contains the tools required to inspect and repair GjallarOS
systems and support installation workflows.

## Installed recovery support

    system/recovery/default.nix

This configures recovery functionality available from the installed system.

## Recovery scripts

    scripts/recovery/

These contain recovery-specific helper and contract scripts.

They are intentionally separate from the main interactive installer.

## Recovery versus installer

The distinction is:

    recovery environment
        bootable maintenance / installer operating system

    gjallar-installer
        interactive installation application

The recovery environment can launch or support installer work, but it is not a
second copy of the installer architecture.

## Security boundary

Recovery can interact with disks and installed systems, so destructive
operations must remain behind explicit validation and confirmation boundaries.

Recovery scripts should not grow independent ad-hoc installation logic when
that responsibility belongs in the main installer/platform installation layer.
