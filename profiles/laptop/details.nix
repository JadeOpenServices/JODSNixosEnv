{ ... }:
{
  # A portable fallback for laptop panels. Hyprland's preferred-mode fallback
  # keeps this valid across internal panels with different resolutions.
  hyprlandMonitors = [
    "eDP-1,preferred,auto,1"
  ];
  monitorsPosition = {
    left = "eDP-1";
    center = "eDP-1";
    right = "eDP-1";
  };
}
