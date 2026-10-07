# Local packages under the `_` prefix, kept apart from nixpkgs names. The
# other .nix files here are picked up by pkgs/lib/overlays.nix as well.
final: prev: {
  _nemu = final.callPackage ../nemu.nix { };
}
