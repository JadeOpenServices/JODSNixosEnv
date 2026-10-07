# gjallar.apps.<id>.enable for every app with a NixOS side.
{ lib, ... }:

let
  catalogue = import ./.;
in
{
  options.gjallar.apps = lib.mapAttrs (_: app: {
    enable = lib.mkEnableOption app.name;
  }) (lib.filterAttrs (_: app: builtins.pathExists (app.path + "/nixos.nix")) catalogue);
}
