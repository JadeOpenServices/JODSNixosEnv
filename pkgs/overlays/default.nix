# Local packages under the `_` prefix, kept apart from nixpkgs names. The
# other .nix files here are picked up by pkgs/lib/overlays.nix as well.
final: prev: {
  # nemu with its VMs and database under XDG data instead of ~/nemu_vm and
  # ~/.nemu.db, and the SPICE viewer path its sample config expects in /usr/bin.
  _nemu = prev.nemu.overrideAttrs (old: {
    cmakeFlags = old.cmakeFlags ++ [
      "-DNM_DEFAULT_VMDIR=.local/share/nemu/vms"
      "-DNM_DEFAULT_DBFILE=.local/share/nemu/nemu.db"
    ];
    postPatch = old.postPatch + ''
      substituteInPlace nemu.cfg.sample \
        --replace-fail /usr/bin/remote-viewer ${final.virt-viewer}/bin/remote-viewer
    '';
  });
}
