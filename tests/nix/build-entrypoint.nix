{ pkgs }:
let
  allowed =
    context:
    builtins.all (a: a.assertion)
      (import ../../system/security/build-entrypoint.nix { buildContext = context; }).assertions;
in
assert !(allowed null);
assert
  !(allowed {
    schemaVersion = 1;
    entrypoint = "nixos-rebuild";
  });
assert
  !(allowed {
    schemaVersion = 2;
    entrypoint = "gjallarctl";
  });
assert allowed {
  schemaVersion = 1;
  entrypoint = "gjallarctl";
};
pkgs.runCommand "gjallar-build-entrypoint-check" { } "touch $out"
