{ config, pkgs, ... }:

{
  wayland.windowManager.hyprland.settings = {
    env = [
      "HYPRCURSOR_THEME,phinger-cursors-light"
      "HYPRCURSOR_SIZE,32"
      "XDG_CURRENT_DESKTOP,Hyprland"
      "XDG_SESSION_TYPE,wayland"
      "XDG_SESSION_DESKTOP,Hyprland"
      "XDG_DATA_DIRS,/etc/profiles/per-user/${config.home.username}/share:/run/current-system/sw/share"
      "QT_QPA_PLATFORM,wayland"
      "QT_WAYLAND_DISABLE_WINDOWDECORATION,1"
      "QT_AUTO_SCREEN_SCALE_FACTOR,1"
      "MOZ_ENABLE_WAYLAND,1"
    ];
  };
}
