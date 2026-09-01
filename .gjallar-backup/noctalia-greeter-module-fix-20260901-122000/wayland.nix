{ pkgs, settings, ... }:

{
    environment.systemPackages = with pkgs; [
        wayland
        wl-clipboard
        bibata-cursors
    ];

    services.xserver = {
        enable = true;

        xkb = {
            variant = settings.keyboardVariant;
            layout = settings.keyboardLayout;
            options = "grp:win_space_toggle";
        };
    };

    programs.noctalia-greeter = {
        enable = true;

        settings = {
            appearance.scheme = "Synced";

            cursor = {
                theme = "Bibata-Modern-Classic";
                size = 24;
                path = "${pkgs.bibata-cursors}/share/icons";
            };

            keyboard = {
                layout = settings.keyboardLayout;
                variant = settings.keyboardVariant;
                options = "grp:win_space_toggle";
            };
        };
    };

    security.polkit.enable = true;
}
