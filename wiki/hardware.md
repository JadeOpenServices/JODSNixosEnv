# Hardware and ODDC

GjallarOS does not select machine profiles.

Hardware discovery records runtime facts, while ODDC resolves canonical device
identity, component composition, capabilities, validated quirks, and exceptional
hardware policy. Generic system modules consume that resolved contract.

Machine-local NixOS hardware discovery is stored in:

    generated/hardware.nix

Graphics and Wi-Fi discovery identify the hardware and active drivers without
turning vendor or PCI identifiers into policy. AMD, Intel, and NVIDIA use generic
graphics support unless ODDC declares a validated device-specific requirement.

Laptop behavior such as battery handling remains generic. Device-specific
capabilities such as charge-threshold support, platform profiles, USB4, embedded
controller support, or validated thermal policy come from ODDC.

ODDC model and component data lives under:

    oddc/catalog/

Generated runtime observations do not become canonical ODDC facts unless they
are intentionally validated as part of the device contract.
