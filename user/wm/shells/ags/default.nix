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
    weather = pkgs.writeShellScriptBin "gjallar-weather" ''
        set -eu
        cache_dir="''${XDG_CACHE_HOME:-$HOME/.cache}/gjallar-weather"
        cache_file="$cache_dir/frankfurt.json"
        ${pkgs.coreutils}/bin/mkdir -p "$cache_dir"

        if response="$(${pkgs.curl}/bin/curl --fail --silent --show-error --max-time 10 \
            'https://api.open-meteo.com/v1/forecast?latitude=50.1109&longitude=8.6821&current=temperature_2m,weather_code&temperature_unit=celsius')"; then
            ${pkgs.coreutils}/bin/printf '%s\n' "$response" > "$cache_file"
        elif [ -r "$cache_file" ]; then
            response="$(${pkgs.coreutils}/bin/cat "$cache_file")"
        else
            ${pkgs.coreutils}/bin/printf '%s\n' '{"text":"󰖐 --°C","tooltip":"Frankfurt am Main weather unavailable","class":"unavailable"}'
            exit 0
        fi

        temperature="$(${pkgs.jq}/bin/jq -r '.current.temperature_2m | round' <<< "$response")"
        code="$(${pkgs.jq}/bin/jq -r '.current.weather_code' <<< "$response")"
        case "$code" in
            0) icon='󰖙'; description='Clear sky' ;;
            1|2) icon='󰖕'; description='Partly cloudy' ;;
            3) icon='󰖐'; description='Overcast' ;;
            45|48) icon='󰖑'; description='Fog' ;;
            51|53|55|56|57) icon='󰖗'; description='Drizzle' ;;
            61|63|65|66|67|80|81|82) icon='󰖖'; description='Rain' ;;
            71|73|75|77|85|86) icon='󰼶'; description='Snow' ;;
            95|96|99) icon='󰖓'; description='Thunderstorm' ;;
            *) icon='󰖐'; description='Weather' ;;
        esac
        ${pkgs.coreutils}/bin/printf \
            '{"text":"%s %s°C","tooltip":"Frankfurt am Main: %s°C — %s","class":"weather"}\n' \
            "$icon" "$temperature" "$temperature" "$description"
    '';
in {
    imports = [ inputs.ags.homeManagerModules.default ];
    home.packages = with pkgs; [
        asztal
        fuzzel
        weather
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

    # Fuzzel reads desktop entries and icons from the Hyprland XDG data paths.
    programs.fuzzel = {
        enable = true;
        settings.main = {
            icons-enabled = true;
            icon-theme = details.icons;
        };
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
            modules-right = [ "custom/weather" "idle_inhibitor" "pulseaudio" "network" "battery" "tray" ];
            clock.format = "{:%a %d %b  %H:%M}";
            network.format-wifi = "  {signalStrength}%";
            network.format-ethernet = "󰈀  connected";
            network.format-disconnected = "󰤮  offline";
            pulseaudio.format = "  {volume}%";
            battery.format = "{icon}  {capacity}%";
            battery.format-icons = [ "󰁺" "󰁼" "󰁾" "󰂀" "󰁹" ];
            "idle_inhibitor" = {
                format = "{icon}";
                format-icons = {
                    activated = "☕";
                    deactivated = "󰾪";
                };
                tooltip-format-activated = "Coffee mode: display stays on";
                tooltip-format-deactivated = "Coffee mode: display may sleep";
            };
            "custom/weather" = {
                exec = "${weather}/bin/gjallar-weather";
                return-type = "json";
                interval = 900;
                tooltip = true;
            };
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
          #clock, #custom-weather, #idle_inhibitor, #pulseaudio, #network, #battery, #tray {
            padding: 0 10px;
          }
          #idle_inhibitor.activated {
            color: #${config.lib.stylix.colors.base0D};
          }
        '';
    };

    home.file.".cache/ags/options-nix.json".text = (builtins.toJSON agsOptions);
}
