{ lib, ... }:
{
  # Stylix currently targets the post-26.05 ReGreet namespace. NixOS 26.05
  # has the same settings under programs.regreet, so provide the namespace
  # expected by Stylix without changing the pinned release.
  options.services.displayManager.regreet = lib.mkOption {
    default = {};
    type = lib.types.submodule {
      options = {
        enable = lib.mkOption { type = lib.types.bool; default = false; };
        package = lib.mkOption { type = lib.types.nullOr lib.types.package; default = null; };
        cageArgs = lib.mkOption { type = lib.types.listOf lib.types.str; default = []; };
        theme = lib.mkOption { type = lib.types.attrs; default = {}; };
        extraCss = lib.mkOption { type = lib.types.lines; default = ""; };
        settings = lib.mkOption { type = lib.types.attrs; default = {}; };
        font = lib.mkOption { type = lib.types.attrs; default = {}; };
        cursorTheme = lib.mkOption { type = lib.types.attrs; default = {}; };
        iconTheme = lib.mkOption { type = lib.types.attrs; default = {}; };
      };
    };
  };
  options.services.kmscon.config = lib.mkOption {
    default = {};
    type = lib.types.attrs;
  };
}
