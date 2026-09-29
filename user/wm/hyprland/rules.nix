{ lib, settings, ... }:

{
  wayland.windowManager.hyprland.settings = {
    windowrule = [
      # USB guard dialogs: float centered, sized relative to the monitor.
      "match:title ^(USB device review|Permanent USB trust|Authorize USB device|USB decision failed)$, float on, center on, size (monitor_w*0.4) (monitor_h*0.5), max_size (monitor_w*0.8) (monitor_h*0.85)"
    ]
    ++ lib.optionals (settings.nextcloudEnable or false) [
      "match:class ^([Nn]extcloud|com\\.nextcloud\\.desktopclient\\.nextcloud)$, focus_on_activate on"
    ];

    workspace = [
      "special,gapsin:24,gapsout:64"
    ];
  };
}
