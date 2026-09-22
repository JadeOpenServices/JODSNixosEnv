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
    targets.gtk.extraCss = ''
      @define-color theme_selected_bg_color #${config.lib.stylix.colors.base0D};
      @define-color theme_selected_fg_color #${config.lib.stylix.colors.base00};

      @define-color theme_bg_color #${config.lib.stylix.colors.base00};
      @define-color theme_fg_color #${config.lib.stylix.colors.base05};
      @define-color theme_base_color #${config.lib.stylix.colors.base00};
      @define-color theme_text_color #${config.lib.stylix.colors.base05};
      @define-color window_bg_color #${config.lib.stylix.colors.base00};
      @define-color window_fg_color #${config.lib.stylix.colors.base05};
      @define-color view_bg_color #${config.lib.stylix.colors.base00};
      @define-color view_fg_color #${config.lib.stylix.colors.base05};
      @define-color headerbar_bg_color #${config.lib.stylix.colors.base01};
      @define-color headerbar_fg_color #${config.lib.stylix.colors.base05};
      @define-color popover_bg_color #${config.lib.stylix.colors.base01};
      @define-color popover_fg_color #${config.lib.stylix.colors.base05};
      @define-color card_bg_color #${config.lib.stylix.colors.base01};
      @define-color card_fg_color #${config.lib.stylix.colors.base05};
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

    targets.ghostty.enable = false;
    targets.tmux.enable = false;
    targets.hyprlock.enable = false;
    targets.hyprland.enable = false;
    targets.btop.enable = lib.mkIf (settings.themeDetails.btopTheme != null) false;
    targets.sway.useWallpaper = false;
  };
}
