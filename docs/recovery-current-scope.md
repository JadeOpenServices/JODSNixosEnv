# Current recovery authority scope

This document freezes the recovery authority surface while the fresh
GjallarOS installation pipeline is being proven.

## Current commands

The supported JODS-facing recovery modes are exactly:

- `gjallar-recover jods repair`
- `gjallar-recover jods reinstall`
- `gjallar-recover jods fresh`

No additional remote wipe, arbitrary repartition, factory-reset, or storage
maintenance command is introduced by the current work.

## Device configuration handoff

`/run/gjallarOS/device-config/user.config.json` is a provider-neutral
configuration input only.

Supplying this file does not grant authority to:

- select a target disk;
- bypass destructive confirmation;
- bypass fresh-install authority;
- access LUKS secrets;
- change Secure Boot policy;
- invoke arbitrary repartitioning;
- add a new recovery operation.

Future JODS support may materialize the device configuration at this path, but
the producer/storage side is intentionally deferred.

## Current goal

The present branch is for proving the unfinished GjallarOS fresh-install
pipeline in a disposable VM:

GPT -> LUKS2 -> Btrfs -> nixos-install -> bootable GjallarOS.

Recovery functionality should not be expanded further until that base path is
working and validated.
