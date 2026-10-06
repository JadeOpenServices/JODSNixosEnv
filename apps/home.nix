# Home Manager side of the app catalogue: every app's home.nix and its
# gjallar.apps.<id>.enable option.
{ lib, ... }:

let
  catalogue = import ./.;
in
{
  imports = lib.concatMap (
    app: lib.optional (builtins.pathExists (app.path + "/home.nix")) (app.path + "/home.nix")
  ) (builtins.attrValues catalogue);

  options.gjallar.apps = lib.mapAttrs (_: app: {
    enable = lib.mkEnableOption app.name;
  }) catalogue;
}
