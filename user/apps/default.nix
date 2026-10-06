{ lib, settings, ... }:

let
  catalogue = import ../../apps;

  # `settings.apps` selects catalogue apps; without it an install keeps
  # every app it had before the selection existed.
  selected = settings.apps or (builtins.attrNames catalogue);

  unknown = builtins.filter (id: !(catalogue ? ${id})) selected;
in
{
  imports = [
    ../../apps/home.nix

    # Base: the window managers launch ghostty, and fingerprint follows
    # the hardware ODDC resolved.
    ./ghostty.nix
    ./fingerprint.nix
  ];

  gjallar.apps =
    assert lib.assertMsg (unknown == [ ]) "unknown apps: ${toString unknown}";
    lib.genAttrs selected (_: {
      enable = true;
    });
}
