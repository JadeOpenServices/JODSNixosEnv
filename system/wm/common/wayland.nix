{ config, pkgs, ... }:

{
    environment.systemPackages = with pkgs; [
        wayland
        wl-clipboard
    ];

    # Configure xwayland
    services.xserver = {
        enable = true;
        xkb = {
            variant = "";
            layout = "us,ru";
            options = "grp:win_space_toggle";
        };
    };

    # SDDM provides a graphical, highly themeable login screen while keeping
    # Hyprland as the only supported desktop session. The default SDDM theme
    # is intentionally used here; OS theming remains controlled by Stylix.
    services.displayManager = {
        defaultSession = "hyprland";
        sddm = {
            enable = true;
            wayland.enable = true;
            # The package's actual installed theme identifier includes the
            # palette accent. Without this exact name SDDM silently falls
            # back to its stock theme.
            theme = "catppuccin-mocha-mauve";
            extraPackages = [ pkgs.catppuccin-sddm ];
        };
    };
}
