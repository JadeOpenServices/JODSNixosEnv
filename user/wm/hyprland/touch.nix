{
  pkgs,
  lib,
  settings,
  hyprlandShellDetails,
  ...
}:
let
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
  shell = hyprlandShellDetails.binds;
  # nixos-26.05 ships hyprgrass from 2025-10-08, which no longer compiles
  # against its Hyprland 0.55.4 (ConfigDataValues.hpp is gone). d094a3e is
  # upstream's hyprpm.toml pin for v0.55.4; drop this once nixpkgs catches up.
  hyprgrass = pkgs.hyprlandPlugins.hyprgrass.overrideAttrs {
    version = "0.8.2-unstable-2026-06-10";
    src = pkgs.fetchFromGitHub {
      owner = "horriblename";
      repo = "hyprgrass";
      rev = "d094a3e62f6ecaeb41515982d3e13edefaf8a4e7";
      hash = "sha256-tCt7FNc1RBHou/ym7B0XzoOqqNq8Df+dizEDkAgJ4U0=";
    };
  };
in
# Touchscreen machines (the installer sets touchscreenEnable when it finds
# one): on-screen keyboard and multi-finger gestures through hyprgrass.
# Hyprland itself only knows a one-finger edge swipe on touchscreens.
# Two-finger swipes and pinches stay with the apps (scroll, zoom).
lib.mkIf (settings.touchscreenEnable or false) {
  home.packages = [
    virtualKeyboard
    pkgs.wvkbd
  ];

  wayland.windowManager.hyprland = {
    plugins = [ hyprgrass ];

    settings = {
      bind = [ "SUPER, K, exec, ${lib.getExe virtualKeyboard}" ];

      exec-once = [ "${pkgs.wvkbd}/bin/wvkbd-mobintl --hidden -H 320 -L 240" ];

      plugin.touch_gestures = {
        # The default is meant for small phone screens; 4.0 is upstream's
        # advice for tablets.
        sensitivity = 4.0;
        # Three fingers left/right switch workspaces.
        workspace_swipe_fingers = 3;
        # Edge swipes are bound below, so none of them switches workspaces.
        workspace_swipe_edge = "none";
        # Wider than the 10 px default, so a finger at the bezel still hits it.
        edge_margin = 20;
      };

      hyprgrass-bind = [
        # Up from the bottom edge: on-screen keyboard.
        ", edge:d:u, exec, ${lib.getExe virtualKeyboard}"
        # Down from the top edge: control center.
        ", edge:u:d, exec, ${shell.controlCenter}"
        # Four fingers up: launcher.
        ", swipe:4:u, exec, ${shell.launcher}"
        # Four fingers left/right: take the window to the next/previous workspace.
        ", swipe:4:l, movetoworkspace, +1"
        ", swipe:4:r, movetoworkspace, -1"
        # Three-finger tap: toggle floating.
        ", tap:3, togglefloating, active"
      ];

      # Hold two fingers on a window and drag it.
      hyprgrass-bindm = [ ", longpress:2, movewindow" ];
    };
  };
}
