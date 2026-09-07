{
  config,
  settings,
  lib,
  ...
}:
let
  details = settings.themeDetails;
in
{
  home.pointerCursor.enable = true;
  xdg.userDirs.setSessionVariables = true;

  gtk = {
    enable = true;
    iconTheme = {
      name = lib.mkForce details.icons;
      package = lib.mkForce details.iconsPkg;
    };

  };

  stylix = {
    # GjallarOS accessibility/contrast layer.
    #
    # Stylix owns GTK theming, so custom GTK CSS must be injected through the
    # Stylix GTK target rather than gtk.gtk3/gtk.gtk4.extraCss.
    targets.gtk.extraCss = ''
      @define-color theme_selected_bg_color #${config.lib.stylix.colors.base0D};
      @define-color theme_selected_fg_color #${config.lib.stylix.colors.base00};
      selection,
      entry selection,
      textview text selection,
      treeview.view:selected,
      row:selected,
      list row:selected {
        background-color: #${config.lib.stylix.colors.base0D};
        color: #${config.lib.stylix.colors.base00};
      }

      *:focus-visible {
        outline-color: #${config.lib.stylix.colors.base0D};
      }
    '';

    targets.nixvim.enable = true;
    # Noctalia writes Kitty's live palette through its template engine.
    targets.kitty.enable = false;
    targets.tmux.enable = false;
    targets.hyprlock.enable = false;
    targets.hyprland.enable = false;
    targets.btop.enable = lib.mkIf (settings.themeDetails.btopTheme != null) false;
    targets.sway.useWallpaper = false;
  };
}
