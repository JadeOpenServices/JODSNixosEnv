{ inputs, pkgs, settings, lib, config, ... }: let
    details = settings.themeDetails;
    asztal = pkgs.callPackage ../../../../non-nix/ags/default.nix
        {inherit inputs;};
    agsColors = {
        wallpaper = details.wallpaper.center;
        theme = {
            blur = (1 - details.opacity) * 100;
            radius = details.rounding;
            shadows = details.shadow;
            palette = {
                primary = {
                    bg = "#${config.lib.stylix.colors.base0D}";
                    fg = "#${config.lib.stylix.colors.base00}";
                };
                secondary = {
                    bg = "#${config.lib.stylix.colors.base0E}";
                    fg = "#${config.lib.stylix.colors.base00}";
                };
                error = {
                    bg = "#${config.lib.stylix.colors.base06}";
                    fg = "#${config.lib.stylix.colors.base00}";
                };
                bg = "#${config.lib.stylix.colors.base00}";
                fg = "#${config.lib.stylix.colors.base05}";
                widget = "#${config.lib.stylix.colors.base02}";
                border = "#${config.lib.stylix.colors.base02}";
            };
        };
        font = {
            size = settings.themeDetails.fontSize;
            name = settings.themeDetails.font;
        };
        widget = {
            opacity = details.opacity * 100;
        };
    };
    agsOptions = lib.recursiveUpdate agsColors details.ags;
in {
    imports = [ inputs.ags.homeManagerModules.default ];
    home.packages = with pkgs; [
        asztal
        fuzzel
        bun
        fd
        dart-sass
        gtk3
        pulsemixer
        networkmanager
        pavucontrol

        brightnessctl
        hyprlock
        sway-contrib.grimshot
        # Spotifyd is slow with playerctl, use dbus instead.
        (pkgs.writeScriptBin "hyprmusic" ''
          #!/bin/sh
          set -euo pipefail
          case "''${1:-}" in
            next) MEMBER=Next ;;
            previous) MEMBER=Previous ;;
            play) MEMBER=Play ;;
            pause) MEMBER=Pause ;;
            play-pause) MEMBER=PlayPause ;;
            *) echo "Usage: $0 next|previous|play|pause|play-pause"; exit 1 ;;
          esac
          exec dbus-send --print-reply \
            --dest="org.mpris.MediaPlayer2.''$(playerctl -l | head -n 1)" \
            /org/mpris/MediaPlayer2 "org.mpris.MediaPlayer2.Player.$MEMBER"
        '')
    ];

    programs.ags = {
        enable = true;
        configDir = ../../../../non-nix/ags;
    };

    programs.waybar = {
        enable = true;
        systemd.enable = false;
        settings.mainBar = {
            layer = "top";
            position = "top";
            height = 30;
            modules-left = [ "hyprland/workspaces" ];
            modules-center = [ "clock" ];
            modules-right = [ "pulseaudio" "network" "battery" "tray" ];
            clock.format = "{:%a %d %b  %H:%M}";
            network.format-wifi = "  {signalStrength}%";
            network.format-ethernet = "󰈀  connected";
            network.format-disconnected = "󰤮  offline";
            pulseaudio.format = "  {volume}%";
            battery.format = "{icon}  {capacity}%";
            battery.format-icons = [ "󰁺" "󰁼" "󰁾" "󰂀" "󰁹" ];
        };
        style = ''
          * {
            font-family: "${settings.themeDetails.font}";
            font-size: ${toString settings.themeDetails.fontSize}px;
          }
          window#waybar {
            background: #${config.lib.stylix.colors.base00};
            color: #${config.lib.stylix.colors.base05};
          }
          #workspaces button {
            color: #${config.lib.stylix.colors.base05};
            padding: 0 8px;
          }
          #workspaces button.active {
            color: #${config.lib.stylix.colors.base0D};
          }
          #clock, #pulseaudio, #network, #battery, #tray {
            padding: 0 10px;
          }
        '';
    };

    home.file.".cache/ags/options-nix.json".text = (builtins.toJSON agsOptions);
}
