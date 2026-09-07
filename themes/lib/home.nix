{
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

    # GjallarOS accessibility/contrast layer.
    #
    # Stylix receives the active Noctalia palette during rebuild, therefore
    # these colors remain theme-derived rather than being hardcoded.
    #
    # base0D = primary/accent
    # base00 = contrasting text on the accent
    gtk3.extraCss = ''
      @define-color theme_selected_bg_color #${config.lib.stylix.colors.base0D};
      @define-color theme_selected_fg_color #${config.lib.stylix.colors.base00};
      @define-color accent_bg_color #${config.lib.stylix.colors.base0D};
      @define-color accent_fg_color #${config.lib.stylix.colors.base00};

      selection,
      entry selection,
      textview text selection,
      treeview.view:selected,
      row:selected {
        background-color: #${config.lib.stylix.colors.base0D};
        color: #${config.lib.stylix.colors.base00};
      }

      *:focus-visible {
        outline-color: #${config.lib.stylix.colors.base0D};
      }
    '';

    gtk4.extraCss = ''
      @define-color theme_selected_bg_color #${config.lib.stylix.colors.base0D};
      @define-color theme_selected_fg_color #${config.lib.stylix.colors.base00};
      @define-color accent_bg_color #${config.lib.stylix.colors.base0D};
      @define-color accent_fg_color #${config.lib.stylix.colors.base00};

      selection,
      entry selection,
      textview text selection,
      row:selected {
        background-color: #${config.lib.stylix.colors.base0D};
        color: #${config.lib.stylix.colors.base00};
      }

      *:focus-visible {
        outline-color: #${config.lib.stylix.colors.base0D};
      }
    '';
  };

  stylix = {
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
