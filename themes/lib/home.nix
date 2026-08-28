{pkgs, settings, lib, ...}: let
    details = settings.themeDetails;
in {
    home.pointerCursor.enable = true;
    xdg.userDirs.setSessionVariables = true;

    gtk = {
        enable = true;
        iconTheme = {
            name = lib.mkForce details.icons;
            package = lib.mkForce details.iconsPkg;
        };
        theme = {
            name = lib.mkForce "catppuccin-frappe-blue-standard";
            package = lib.mkForce pkgs.catppuccin-gtk;
        };
    };

    stylix = {
        targets.nixvim.enable =
            lib.mkIf (settings.themeDetails.themeName != null) false;
        targets.tmux.enable = false;
        targets.hyprlock.enable = false;
        targets.hyprland.enable = false;
        targets.btop.enable =
            lib.mkIf (settings.themeDetails.btopTheme != null) false;
        targets.sway.useWallpaper = false;
    };
}
