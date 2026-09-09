# Installer device configuration handoff

GjallarOS recovery and fresh-install media expose one provider-neutral runtime
configuration contract.

## Runtime input

A trusted provisioning component may materialize:

`/run/gjallarOS/device-config/user.config.json`

before invoking the fresh/reinstall recovery flow.

The producer may eventually be:

- JODS device recovery/reset orchestration;
- GjallarOS recovery storage restoring preserved per-device configuration;
- another explicitly trusted provisioning workflow.

GjallarOS owns validation and consumption of this file.

## Precedence

1. `/run/gjallarOS/device-config/user.config.json`
2. embedded `/etc/gjallar/installer-fallback-user.config.json`
3. normal interactive installer fallback

The embedded fallback reuses the device `user.config.json` present in the exact
flake source used to build the recovery image. If no source user config exists,
the image synthesizes a nonsecret fallback from its build-time settings.

## Security boundary

Configuration input does not authorize storage mutation.

The normal GjallarOS installer still owns:

- target-disk validation;
- destructive-plan confirmation;
- fresh-install authority;
- LUKS credential handling;
- Secure Boot checks;
- recovery partition policy;
- installation-complete lifecycle state.

The handoff must not contain plaintext passwords, LUKS keys, TPM secrets,
private signing keys, or enrollment secrets.

## Future JODS dependency

A future JODS reset/wipe workflow only needs to authenticate its recovery
operation and materialize the device's stored configuration at the runtime
input path before invoking the existing GjallarOS recovery flow.

No JODS producer implementation is part of GJAL-65.
