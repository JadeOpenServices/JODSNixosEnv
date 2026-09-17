{
  config,
  pkgs,
  settings,
  inputs,
  lib,
  hyprlandShellDetails,
  ...
}:
let
  gjallarPasteOnce = pkgs.writeShellScript "gjallar-paste-once" ''
    set -eu

    case "''${1:-}" in
      ctrl-v)
        ${pkgs.hyprland}/bin/hyprctl dispatch sendshortcut CTRL,V,activewindow >/dev/null
        ;;
      ctrl-shift-v)
        ${pkgs.hyprland}/bin/hyprctl dispatch sendshortcut "CTRL SHIFT",V,activewindow >/dev/null
        ;;
      shift-insert)
        ${pkgs.hyprland}/bin/hyprctl dispatch sendshortcut SHIFT,Insert,activewindow >/dev/null
        ;;
      *)
        exit 2
        ;;
    esac

    # Give the focused client enough time to consume the selection before
    # destroying it. Keep this short enough to feel immediate.
    ${pkgs.coreutils}/bin/sleep 0.12

    # Remove both the normal clipboard and primary selection.
    ${pkgs.wl-clipboard}/bin/wl-copy --clear >/dev/null 2>&1 || true
    ${pkgs.wl-clipboard}/bin/wl-copy --primary --clear >/dev/null 2>&1 || true
  '';
  themeDetails = settings.themeDetails;
  shellDetails = hyprlandShellDetails;
  wallpaperDetails =
    if builtins.isAttrs themeDetails.wallpaper then
      themeDetails.wallpaper
    else
      { center = themeDetails.wallpaper; };
  startupWallpaper =
    if settings.backgroundNormal != "" then
      settings.backgroundNormal
    else
      wallpaperDetails.center;
  sessionStart = pkgs.writeShellScript "gjallar-hyprland-session-start" ''
    # Noctalia stores the wallpaper selected in its UI here.  Use the same
    # image immediately, rather than briefly showing the Nix fallback first.
    selected_wallpaper=${lib.escapeShellArg startupWallpaper}
    noctalia_state="''${XDG_STATE_HOME:-$HOME/.local/state}/noctalia/settings.toml"
    if [ -r "$noctalia_state" ]; then
      noctalia_wallpaper="$(${pkgs.gawk}/bin/awk '
        /^\[wallpaper\.last\]$/ { in_last = 1; next }
        /^\[/ { in_last = 0 }
        in_last && /^path[[:space:]]*=/ {
          sub(/^[^=]*=[[:space:]]*/, "")
          gsub(/^"|"$/, "")
          print
          exit
        }
      ' "$noctalia_state")"
      if [ -n "$noctalia_wallpaper" ] && [ -r "$noctalia_wallpaper" ]; then
        selected_wallpaper="$noctalia_wallpaper"
      fi
    fi
    ${pkgs.swaybg}/bin/swaybg --image "$selected_wallpaper" --mode fill &
    ${pkgs.coreutils}/bin/sleep 0.1
    # Give NetworkManager a brief chance to establish an actual connection.
    # This prevents Noctalia weather/plugin/API refreshes from racing early boot.
    # The timeout is bounded: offline use must never prevent the shell starting.
    ${pkgs.networkmanager}/bin/nm-online -q -t 10 || true

    exec ${lib.getExe inputs.noctalia.packages.${pkgs.stdenv.hostPlatform.system}.default}
  '';
  virtualKeyboard = pkgs.writeShellApplication {
    name = "gjallar-virtual-keyboard";
    runtimeInputs = [
      pkgs.procps
      pkgs.wvkbd
    ];
    text = ''
      set -euo pipefail
      if pgrep -x wvkbd-mobintl >/dev/null; then
        pkill -USR2 -x wvkbd-mobintl
      else
        ${pkgs.wvkbd}/bin/wvkbd-mobintl --hidden -H 320 -L 240 >/dev/null 2>&1 &
        disown
        sleep 0.1
        pkill -USR2 -x wvkbd-mobintl
      fi
    '';
  };
in
{
  home.packages =
    (with pkgs; [
      awww
      swaybg
      wayvnc
    ])
    ++ lib.optionals (settings.touchscreenEnable or false) [
      virtualKeyboard
      pkgs.wvkbd
    ];

  wayland.windowManager.hyprland.settings = {
    bind =
      lib.optionals config.programs.noctalia.enable [
        "SUPER, C, exec, ${lib.getExe config.programs.noctalia.package} msg status >/dev/null 2>&1 && ${lib.getExe config.programs.noctalia.package} msg panel-toggle control-center >/dev/null 2>&1 || true"
      ]
      ++ lib.optionals (settings.touchscreenEnable or false) [
        "SUPER, K, exec, ${lib.getExe virtualKeyboard}"

        "CTRL, V, exec, ${gjallarPasteOnce} ctrl-v"
        "CTRL SHIFT, V, exec, ${gjallarPasteOnce} ctrl-shift-v"
        "SHIFT, INSERT, exec, ${gjallarPasteOnce} shift-insert"
      ];

    monitor = [
      ",preferred,auto,1"
    ];

    exec-once =
      lib.optionals config.programs.noctalia.enable [
        "${sessionStart}"
      ]
      ++ lib.optionals (settings.touchscreenEnable or false) [
        "${pkgs.wvkbd}/bin/wvkbd-mobintl --hidden -H 320 -L 240"
      ];

    general = {
      gaps_in = 8;
      gaps_out = 16;
      border_size = 2;
      allow_tearing = true;
    };

    cursor = {
      inactive_timeout = 5;
    };

    decoration = {
      dim_special = 0.5;
      rounding = themeDetails.rounding;

      blur = {
        enabled = true;
        special = false;
        brightness = 1.0;
        contrast = 1.0;
        noise = 0.02;
        passes = 3;
        size = 10;
      };

      shadow = {
        enabled = themeDetails.shadow;
        offset = "2 2";
        range = 20;
      };
    };

    animations = {
      enabled = true;

      bezier = [
        "wind, 0.05, 0.9, 0.1, 1.05"
        "winIn, 0.1, 1.1, 0.1, 1.1"
        "winOut, 0.3, -0.3, 0, 1"
        "liner, 1, 1, 1, 1"
        "workIn, 0.72, -0.07, 0.41, 0.98"
      ];

      animation = [
        "windows, 1, 6, wind, slide"
        "windowsIn, 1, 6, winIn, slide"
        "windowsOut, 1, 5, winOut, slide"
        "windowsMove, 1, 5, wind, slide"
        "border, 1, 1, liner"
        "borderangle, 1, 30, liner, loop"
        "fade, 1, 10, default"
        "workspaces, 1, 5, wind"
        "specialWorkspace, 1, 5, workIn, slidevert"
      ];
    };

    debug = {
      disable_logs = false;
    };

    input = {
      kb_layout = settings.keyboardLayout;
      kb_variant = settings.keyboardVariant;
      kb_options = "grp:alt_shift_toggle";
      follow_mouse = true;

      touchpad = {
        natural_scroll = true;
      };

      # Hyprland handles pressure, tilt, erasers, and tablet-pad buttons
      # natively. Pens use absolute positioning by default.
      tablet = lib.mkIf (settings.penTabletEnable or false) {
        relative_input = false;
      };
    };

    # Hyprland 0.55 removed gestures.workspace_swipe. The replacement is the
    # top-level gesture keyword; native libinput remains sufficient here.
    gesture = lib.optionals settings.touchpadWorkspaceSwipe [
      "3, horizontal, workspace"
    ];

    gestures = {
      workspace_swipe_touch = settings.touchscreenEnable or false;
      workspace_swipe_cancel_ratio = 0.15;
      workspace_swipe_forever = true;
      workspace_swipe_distance = 200;
    };

    dwindle = {
      preserve_split = true;
      force_split = 2;
      split_width_multiplier = 1.5;
    };

    misc = {
      force_default_wallpaper = -1;
      disable_hyprland_logo = true;
      disable_splash_rendering = true;
      exit_window_retains_fullscreen = true;
    };
  };
}
