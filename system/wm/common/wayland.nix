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

    services.greetd = {
        enable = true;
        settings.default_session = {
            command = "${pkgs.tuigreet}/bin/tuigreet --time --remember --cmd ${pkgs.hyprland}/bin/Hyprland";
            user = "greeter";
        };
    };
}
