# Home Manager side of the app catalogue: every app's home.nix and its
# gjallar.apps.<id>.enable option.
{ lib, ... }:

let
  catalogue = lib.filterAttrs (_: app: builtins.pathExists (app.path + "/home.nix")) (import ./.);
in
{
  imports = map (app: app.path + "/home.nix") (builtins.attrValues catalogue);

  options.gjallar.apps = lib.mapAttrs (_: app: {
    enable = lib.mkEnableOption app.name;
  }) catalogue;
}
