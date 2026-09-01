{ pkgs, settings, ... }:

let
    catppuccinSddm = pkgs.catppuccin-sddm.override {
        flavor = "mocha";
        accent = "mauve";
        font = settings.themeDetails.font;
        fontSize = toString settings.themeDetails.fontSize;
    };
in {
    environment.systemPackages = with pkgs; [
        wayland
        wl-clipboard
        bibata-cursors
        catppuccinSddm
    ];

    services.xserver = {
        enable = true;
        xkb = {
            variant = settings.keyboardVariant;
            layout = settings.keyboardLayout;
            options = "grp:win_space_toggle";
        };
    };

    # The packaged theme is installed in SDDM's NixOS theme path.
    services.displayManager = {
        defaultSession = "hyprland";
        sddm = {
            enable = true;
            wayland = { enable = true; compositor = "kwin"; };
            theme = "catppuccin-mocha-mauve";

            settings = {
                Theme = {
                    CursorTheme = "Bibata-Modern-Classic";
                    CursorSize = 24;
                };
            };
        };
    };

    # Apply normal rebuilds live without ending the current graphical session.
    # A changed SDDM unit/theme is picked up on the next login or reboot.
    systemd.services.display-manager.stopIfChanged = false;
}
