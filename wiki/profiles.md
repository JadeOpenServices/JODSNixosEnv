
# Profiles

Profiles describe machine classes and connect them to reusable GjallarOS
modules.

Main path:

    profiles/

Examples currently include:

    laptop/
    desktop/
    thinkpad/
    work/
    work-user/

## Typical profile structure

A machine profile normally contains files such as:

    configuration.nix
    home.nix
    details.nix
    hardware-configuration.nix

## configuration.nix

This is the NixOS entry point for the profile.

It selects or imports the system modules needed by that machine.

## home.nix

This connects the profile to Home Manager and user-level configuration.

## details.nix

This stores profile-specific details that are used by the rest of the
configuration.

## hardware-configuration.nix

This is machine-generated hardware state.

The installer generates it from the machine rather than using it as a place for
general reusable hardware policy.

Reusable hardware behavior belongs under:

    system/hardware/

## Role of a profile

A profile should mostly answer:

    "Which reusable GjallarOS modules make up this type of machine?"

It should not duplicate large application, security, desktop or hardware
implementations that already belong in their respective module trees.
