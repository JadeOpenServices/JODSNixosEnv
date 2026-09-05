{
  settings,
  lib,
  ...
}:
let
  details = settings.themeDetails;
in
{
  home.pointerCursor.enable = true;
  xdg.userDirs.setSessionVariables = true;

  gtk = {
    enable = true;
    iconTheme = {
      name = lib.mkForce details.icons;
      package = lib.mkForce details.iconsPkg;
    };
  };

  stylix = {
    targets.nixvim.enable = true;
    targets.tmux.enable = false;
    targets.hyprlock.enable = false;
    targets.hyprland.enable = false;
    targets.btop.enable = lib.mkIf (settings.themeDetails.btopTheme != null) false;
    targets.sway.useWallpaper = false;
  };
}
