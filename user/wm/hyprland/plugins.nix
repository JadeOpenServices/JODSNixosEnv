{
  config,
  settings,
  lib,
  ...
}:

{
  wayland.windowManager.hyprland.extraConfig = lib.optionalString settings.themeDetails.bordersPlusPlus ''
    plugin {
      borders-plus-plus {
          add_borders = 2

          col.border_1 = rgb(${config.lib.stylix.colors.base00})
          col.border_2 = rgb(${config.lib.stylix.colors.base00})

          border_size_1 = 3
          border_size_2 = 10

          natural_rounding = yes
      }
    }
  '';
}
