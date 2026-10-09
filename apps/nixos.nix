# NixOS side of the app catalogue: every app's nixos.nix and its
# gjallar.apps.<id>.enable option.
{ lib, ... }:

let
  catalogue = import ./.;
in
{
  imports = [
    ./nixos-options.nix
  ]
  ++ lib.concatMap (
    app: lib.optional (builtins.pathExists (app.path + "/nixos.nix")) (app.path + "/nixos.nix")
  ) (builtins.attrValues catalogue);
}
