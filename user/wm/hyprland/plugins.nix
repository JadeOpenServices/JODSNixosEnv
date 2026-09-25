{
  settings,
  lib,
  ...
}:

{
  wayland.windowManager.hyprland.extraConfig = lib.optionalString settings.themeDetails.bordersPlusPlus ''
    plugin {
      borders-plus-plus {
          add_borders = 2

          col.border_1 = $surface
          col.border_2 = $surface

          border_size_1 = 3
          border_size_2 = 10

          natural_rounding = yes
      }
    }
  '';
}
