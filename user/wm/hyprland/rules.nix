{ lib, settings, ... }:

{
  wayland.windowManager.hyprland.settings = {
    windowrule = lib.optionals (settings.nextcloudEnable or false) [
      "match:class ^([Nn]extcloud|com\\.nextcloud\\.desktopclient\\.nextcloud)$, focus_on_activate on"
    ];

    workspace = [
      "special,gapsin:24,gapsout:64"
    ];
  };
}
