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
    menu = pkgs.writeShellScriptBin "gjallar-menu" ''
        set -eu
        title="$1"
        shift
        exec ${pkgs.yad}/bin/yad --list --title="$title" --text="$title" \
            --width=760 --height=520 --center --search-column=1 \
            --separator='|' --print-column=3 \
            --column='Device' --column='Status' --column='Identifier' \
            --button='Cancel:1' --button='Select:0' -- "$@"
    '';
    wifiMenu = pkgs.writeShellScriptBin "gjallar-wifi-menu" ''
        set -u
        nmcli=${pkgs.networkmanager}/bin/nmcli
        menu=${menu}/bin/gjallar-menu
        $nmcli radio wifi on || true
        rows=()
        while IFS=: read -r active ssid signal security; do
          [ -n "$ssid" ] || continue
          if [ "$active" = yes ]; then
            status="Connected · ''${signal:-?}%"
            rows+=("$ssid" "$status" "$ssid")
          fi
        done < <($nmcli -t --escape no -f ACTIVE,SSID,SIGNAL,SECURITY device wifi list --rescan yes 2>/dev/null || true)
        while IFS=: read -r active ssid signal security; do
          [ -n "$ssid" ] || continue
          [ "$active" = yes ] && continue
          status="''${signal:-?}%"
          [ -n "$security" ] && status="$status · secured"
          rows+=("$ssid" "$status" "$ssid")
        done < <($nmcli -t --escape no -f ACTIVE,SSID,SIGNAL,SECURITY device wifi list 2>/dev/null || true)
        if [ ''${#rows[@]} -eq 0 ]; then
          ${pkgs.yad}/bin/yad --info --title='Wi‑Fi' --text='No Wi‑Fi networks found.'
          exit 0
        fi
        selected="$($menu 'Wi‑Fi networks' "''${rows[@]}")" || exit 0
        [ -n "$selected" ] || exit 0
        if $nmcli -t -f ACTIVE,SSID device wifi list | ${pkgs.gnugrep}/bin/grep -Fxq "yes:$selected"; then
          exit 0
        fi
        if ! $nmcli device wifi connect "$selected"; then
          password="$(${pkgs.yad}/bin/yad --entry --hide-text --title='Wi‑Fi password' --text="Password for $selected:" --button='Connect:0' --button='Cancel:1')" || exit 0
          $nmcli device wifi connect "$selected" password "$password"
        fi
    '';
    bluetoothMenu = pkgs.writeShellScriptBin "gjallar-bluetooth-menu" ''
        set -u
        bluetoothctl=${pkgs.bluez}/bin/bluetoothctl
        menu=${menu}/bin/gjallar-menu
        $bluetoothctl power on >/dev/null || true
        $bluetoothctl --timeout 6 scan on >/dev/null 2>&1 || true
        rows=()
        while IFS=' ' read -r _ mac name; do
          [ -n "$mac" ] || continue
          info="$($bluetoothctl info "$mac" 2>/dev/null || true)"
          ${pkgs.gnugrep}/bin/grep -q 'Connected: yes' <<< "$info" || continue
          rows+=("''${name:-$mac}" 'Connected' "$mac")
        done < <($bluetoothctl devices 2>/dev/null || true)
        while IFS=' ' read -r _ mac name; do
          [ -n "$mac" ] || continue
          info="$($bluetoothctl info "$mac" 2>/dev/null || true)"
          ${pkgs.gnugrep}/bin/grep -q 'Connected: yes' <<< "$info" && continue
          if ${pkgs.gnugrep}/bin/grep -q 'Paired: yes' <<< "$info"; then
            rows+=("''${name:-$mac}" 'Paired' "$mac")
          else
            rows+=("''${name:-$mac}" 'Available' "$mac")
          fi
        done < <($bluetoothctl devices 2>/dev/null || true)
        if [ ''${#rows[@]} -eq 0 ]; then
          ${pkgs.yad}/bin/yad --info --title='Bluetooth' --text='No Bluetooth devices found.'
          exit 0
        fi
        selected="$($menu 'Bluetooth devices' "''${rows[@]}")" || exit 0
        [ -n "$selected" ] || exit 0
        info="$($bluetoothctl info "$selected" 2>/dev/null || true)"
        if ${pkgs.gnugrep}/bin/grep -q 'Connected: yes' <<< "$info"; then
          $bluetoothctl disconnect "$selected"
        elif ${pkgs.gnugrep}/bin/grep -q 'Paired: yes' <<< "$info"; then
          $bluetoothctl connect "$selected"
        else
          printf 'agent on\ndefault-agent\npair %s\ntrust %s\nconnect %s\n' "$selected" "$selected" "$selected" | $bluetoothctl
        fi
    '';
    batteryMenu = pkgs.writeShellScriptBin "gjallar-battery-menu" ''
        set -eu
        menu=${menu}/bin/gjallar-menu
        powerprofilesctl=${pkgs.power-profiles-daemon}/bin/powerprofilesctl
        current="$($powerprofilesctl get 2>/dev/null || printf 'balanced')"
        selected="$($menu 'Power profile' \
          "Power saver" "Current: $current" 'power-saver' \
          'Balanced' "Current: $current" 'balanced' \
          'Performance' "Current: $current" 'performance')" || exit 0
        [ -n "$selected" ] && $powerprofilesctl set "$selected"
    '';
    weatherMenu = pkgs.writeShellScriptBin "gjallar-weather-menu" ''
        set -eu
        response="$(${pkgs.curl}/bin/curl --fail --silent --show-error --max-time 10 \
          'https://api.open-meteo.com/v1/forecast?latitude=50.1109&longitude=8.6821&daily=weather_code,temperature_2m_max,temperature_2m_min&temperature_unit=celsius&timezone=Europe%2FBerlin')" || {
          ${pkgs.yad}/bin/yad --error --title='Frankfurt weather' --text='Forecast unavailable. Check your Internet connection.'
          exit 1
        }
        forecast='Frankfurt am Main forecast\n\n'
        count="$(${pkgs.jq}/bin/jq '.daily.time | length' <<< "$response")"
        for ((index=0; index<count; index++)); do
          date="$(${pkgs.jq}/bin/jq -r ".daily.time[$index]" <<< "$response")"
          low="$(${pkgs.jq}/bin/jq -r ".daily.temperature_2m_min[$index] | round" <<< "$response")"
          high="$(${pkgs.jq}/bin/jq -r ".daily.temperature_2m_max[$index] | round" <<< "$response")"
          code="$(${pkgs.jq}/bin/jq -r ".daily.weather_code[$index]" <<< "$response")"
          case "$code" in
            0) condition='Clear' ;;
            1|2) condition='Partly cloudy' ;;
            3) condition='Overcast' ;;
            45|48) condition='Fog' ;;
            51|53|55|56|57) condition='Drizzle' ;;
            61|63|65|66|67|80|81|82) condition='Rain' ;;
            71|73|75|77|85|86) condition='Snow' ;;
            95|96|99) condition='Thunderstorm' ;;
            *) condition='Weather' ;;
          esac
          forecast+="$date  $low–$high°C  $condition\n"
        done
        ${pkgs.yad}/bin/yad --text-info --title='Frankfurt weather' --width=560 --height=380 \
          --button='Close:0' --filename=<(printf '%b' "$forecast")
    '';
    powerMenu = pkgs.writeShellScriptBin "gjallar-power-menu" ''
        set -eu
        menu=${menu}/bin/gjallar-menu
        selected="$($menu 'Power' \
          'Sleep' 'Suspend this PC' 'suspend' \
          'Hibernate' 'Save memory to disk and power off' 'hibernate' \
          'Restart' 'Reboot this PC' 'reboot' \
          'Shutdown' 'Power off this PC' 'poweroff')" || exit 0
        case "$selected" in
          suspend|hibernate|reboot|poweroff)
            ${pkgs.systemd}/bin/systemctl "$selected"
            ;;
        esac
    '';
in {
    imports = [ inputs.ags.homeManagerModules.default ];
    home.packages = with pkgs; [
        asztal
        fuzzel
        weather
        menu
        wifiMenu
        bluetoothMenu
        batteryMenu
        weatherMenu
        powerMenu
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
            modules-right = [ "custom/weather" "idle_inhibitor" "pulseaudio" "bluetooth" "network" "battery" "tray" "custom/power" ];
            "hyprland/workspaces" = {
                format = "{icon}";
                "workspace-taskbar" = {
                    enable = true;
                    "update-active-window" = true;
                    "active-window-position" = "last";
                    format = "{icon}";
                    "icon-size" = 18;
                    "max-icons" = 1;
                    "icon-theme" = [ details.icons ];
                };
            };
            clock.format = "{:%a %d %b  %H:%M}";
            network.format-wifi = "  {signalStrength}%";
            network.format-ethernet = "󰈀  connected";
            network.format-disconnected = "󰤮  offline";
            network.on-click = "${wifiMenu}/bin/gjallar-wifi-menu";
            network.tooltip-format-wifi = "{essid}\nClick for Wi‑Fi networks";
            network.tooltip-format-disconnected = "Click for Wi‑Fi networks";
            bluetooth = {
                format = "";
                format-connected = " {num_connections}";
                format-disabled = "󰂲";
                tooltip-format = "Bluetooth\nClick for devices";
                on-click = "${bluetoothMenu}/bin/gjallar-bluetooth-menu";
            };
            pulseaudio.format = "  {volume}%";
            battery.format = "{icon}  {capacity}%";
            battery.format-icons = [ "󰁺" "󰁼" "󰁾" "󰂀" "󰁹" ];
            battery.on-click = "${batteryMenu}/bin/gjallar-battery-menu";
            battery.tooltip-format = "{capacity}% · {timeTo}\nClick for power profile";
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
                on-click = "${weatherMenu}/bin/gjallar-weather-menu";
            };
            "custom/power" = {
                format = "⏻";
                tooltip = true;
                tooltip-format = "Power menu";
                on-click = "${powerMenu}/bin/gjallar-power-menu";
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
          #workspaces image {
          }
          #clock, #custom-weather, #idle_inhibitor, #pulseaudio, #bluetooth, #network, #battery, #tray, #custom-power {
            padding: 0 10px;
          }
          #idle_inhibitor.activated {
            color: #${config.lib.stylix.colors.base0D};
          }
          #custom-power {
            color: #${config.lib.stylix.colors.base08};
          }
        '';
    };

    home.file.".cache/ags/options-nix.json".text = (builtins.toJSON agsOptions);
}
