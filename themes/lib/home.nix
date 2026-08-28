{pkgs, settings, lib, ...}: let
    details = settings.themeDetails;
in {
    gtk = {
        enable = true;
        iconTheme = {
            name = details.icons;
            package = details.iconsPkg;
        };
        theme = {
            name = "catppuccin-frappe-blue-standard";
            package = pkgs.catppuccin-gtk;
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
        targets.noctalia.enable = false;
    };
}
