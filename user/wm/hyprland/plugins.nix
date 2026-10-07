{
  settings,
  lib,
  pkgs,
  ...
}:
lib.mkIf settings.themeDetails.bordersPlusPlus {
  # Themes that ask for it get a second, wider border ring in the surface
  # colour around every window. $surface comes from the Noctalia colours
  # sourced in extraConfig, so this block has to follow that source line.
  wayland.windowManager.hyprland = {
    plugins = [ pkgs.hyprlandPlugins.borders-plus-plus ];
    extraConfig = lib.mkAfter ''
      plugin:borders-plus-plus:add_borders = 2
      plugin:borders-plus-plus:col.border_1 = $surface
      plugin:borders-plus-plus:col.border_2 = $surface
      plugin:borders-plus-plus:border_size_1 = 3
      plugin:borders-plus-plus:border_size_2 = 10
      plugin:borders-plus-plus:natural_rounding = yes
    '';
  };
}
