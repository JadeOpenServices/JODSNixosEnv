# gjallar.apps.<id>.enable for every app with a NixOS side, selected by
# `settings.apps`.
{ lib, settings, ... }:

let
  catalogue = lib.filterAttrs (_: app: builtins.pathExists (app.path + "/nixos.nix")) (import ./.);
in
{
  options.gjallar.apps = lib.mapAttrs (_: app: {
    enable = lib.mkEnableOption app.name;
  }) catalogue;

  config.gjallar.apps = lib.mapAttrs (id: _: {
    enable = lib.mkDefault (builtins.elem id settings.apps);
  }) catalogue;
}
