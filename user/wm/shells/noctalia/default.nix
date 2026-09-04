{
  pkgs,
  inputs,
  config,
  lib,
  settings,
  ...
}:
let
  themeDetails = settings.themeDetails;
  wallpaperDetails =
    if builtins.isAttrs themeDetails.wallpaper then
      themeDetails.wallpaper
    else
      {
        left = themeDetails.wallpaper;
        center = themeDetails.wallpaper;
        right = themeDetails.wallpaper;
      };
  avatarPath = if themeDetails ? avatar then themeDetails.avatar else wallpaperDetails.center;
  isWorkUser = config.home.username == settings.workUsername;
  selectedBackground =
    if isWorkUser && settings.backgroundWork != "" then
      settings.backgroundWork
    else if !isWorkUser && settings.backgroundNormal != "" then
      settings.backgroundNormal
    else
      wallpaperDetails.center;
  palette = {
    dark = {
      mPrimary = "#${config.lib.stylix.colors.base0D}";
      mOnPrimary = "#${config.lib.stylix.colors.base00}";
      mSecondary = "#${config.lib.stylix.colors.base0A}";
      mOnSecondary = "#${config.lib.stylix.colors.base00}";
      mTertiary = "#${config.lib.stylix.colors.base0C}";
      mOnTertiary = "#${config.lib.stylix.colors.base00}";
      mError = "#${config.lib.stylix.colors.base08}";
      mOnError = "#${config.lib.stylix.colors.base00}";
      mSurface = "#${config.lib.stylix.colors.base00}";
      mOnSurface = "#${config.lib.stylix.colors.base05}";
      mSurfaceVariant = "#${config.lib.stylix.colors.base01}";
      mOnSurfaceVariant = "#${config.lib.stylix.colors.base05}";
      mOutline = "#${config.lib.stylix.colors.base04}";
      mShadow = "#${config.lib.stylix.colors.base00}";
      mHover = "#${config.lib.stylix.colors.base02}";
      mOnHover = "#${config.lib.stylix.colors.base05}";

      terminal = {
        background = "#${config.lib.stylix.colors.base00}";
        foreground = "#${config.lib.stylix.colors.base05}";

        cursor = "#${config.lib.stylix.colors.base05}";
        cursorText = "#${config.lib.stylix.colors.base00}";

        selectionBg = "#${config.lib.stylix.colors.base02}";
        selectionFg = "#${config.lib.stylix.colors.base05}";

        normal = {
          black = "#${config.lib.stylix.colors.base00}";
          red = "#${config.lib.stylix.colors.base08}";
          green = "#${config.lib.stylix.colors.base0B}";
          yellow = "#${config.lib.stylix.colors.base0A}";
          blue = "#${config.lib.stylix.colors.base0D}";
          magenta = "#${config.lib.stylix.colors.base0E}";
          cyan = "#${config.lib.stylix.colors.base0C}";
          white = "#${config.lib.stylix.colors.base05}";
        };

        bright = {
          black = "#${config.lib.stylix.colors.base03}";
          red = "#${config.lib.stylix.colors.base08}";
          green = "#${config.lib.stylix.colors.base0B}";
          yellow = "#${config.lib.stylix.colors.base0A}";
          blue = "#${config.lib.stylix.colors.base0D}";
          magenta = "#${config.lib.stylix.colors.base0E}";
          cyan = "#${config.lib.stylix.colors.base0C}";
          white = "#${config.lib.stylix.colors.base07}";
        };
      };
    };
  };
in
{
  programs.noctalia = {
    enable = true;
    systemd.enable = true;
    package = inputs.noctalia.packages.${pkgs.stdenv.hostPlatform.system}.default;

    settings = lib.recursiveUpdate (import ./settings.nix) {
      corner_radius_scale = 2.0;

      theme = {
        mode = "dark";
        source = "builtin";
        builtin = "Noctalia";
        templates.user.hyprland = {
          input_path = "$XDG_CONFIG_HOME/noctalia/templates/hyprland.conf";
          output_path = "$XDG_STATE_HOME/noctalia/hyprland-colors.conf";
          post_hook = "${pkgs.hyprland}/bin/hyprctl reload; ${lib.getExe inputs.noctalia.packages.${pkgs.stdenv.hostPlatform.system}.default} msg greeter-sync";
        };
        templates.user.superfile = {
          input_path = "$XDG_CONFIG_HOME/noctalia/templates/superfile.toml";
          output_path = "$XDG_CONFIG_HOME/superfile/theme/noctalia.toml";
        };
        templates.user.kitty = {
          input_path = "$XDG_CONFIG_HOME/noctalia/templates/kitty.conf";
          output_path = "$XDG_CONFIG_HOME/kitty/noctalia.conf";
        };
      };
      accessibility = {
        ui_scale = 1.4;
      };
      shell = {
        font_family = themeDetails.font;
        avatar_path = avatarPath;
        screenshot = {
          directory = "~/Media/Pictures/Screenshots";
        };
        panel = {
          clipboard_placement = "attached";
          open_near_click_clipboard = true;
          open_near_click_control_center = true;
        };
      };
      widget = {
        launcher = {
          glyph = "skull";
          scale = 1.25;
        };
        keyboard_layout = {
          show_icon = false;
        };
        media = {
          hide_album_art = true;
        };
        volume = {
          show_label = false;
          scroll_step = 2;
        };
        spacer_default = {
          type = "spacer";
          length = 15;
        };
        workspaces = {
          style = "minimal";
          occupied_color = "on_surface";
          scale = 1.2;
        };
        audio_visualizer = {
          width = 120;
        };
        tray = {
          drawer = true;
        };
      };
      location = {
        auto_locate = false;
        address = "Moscow, RU";
      };
      osd = {
        orientation = "horizontal";
        position = "top_center";
        kinds = {
          keyboard_layout = false;
        };
      };
      bar.default = {
        scale = 1.4;
        thickness = 40;
        margin_edge = 0;
        margin_ends = 0;
        start = [
          "launcher"
          "spacer_default"
          "media"
        ];
        center = [
          "audio_visualizer"
          "workspaces"
          "audio_visualizer"
        ];
        end = [
          "tray"
          "keyboard_layout"
          "notifications"
          "clipboard"
          "volume"
          "battery"
          "spacer_default"
          "clock"
        ];
      };
      wallpaper = {
        directory = "${settings.dotfilesDir}/non-nix/wallpapers";
        fill_mode = "crop";
        transition_on_startup = false;
        # This is only the initial image.  The picker persists later choices
        # in Noctalia's state file without narrowing the browse directory.
        default.path = selectedBackground;
      };
    };
  };

  home.file.".config/noctalia/palettes/stylix.json".text = (builtins.toJSON palette);

  home.file.".config/noctalia/templates/superfile.toml".text = ''
    # Generated by Noctalia. Do not edit: palette changes replace this file.
    code_syntax_highlight = "dracula"

    full_screen_fg = "{{colors.on_surface.default.hex}}"
    full_screen_bg = "{{colors.surface.default.hex}}"
    gradient_color = ["{{colors.primary.default.hex}}", "{{colors.tertiary.default.hex}}"]
    directory_icon_color = "{{colors.primary.default.hex}}"

    file_panel_fg = "{{colors.on_surface.default.hex}}"
    file_panel_bg = "{{colors.surface.default.hex}}"
    file_panel_border = "{{colors.outline.default.hex}}"
    file_panel_border_active = "{{colors.primary.default.hex}}"
    file_panel_top_directory_icon = "{{colors.secondary.default.hex}}"
    file_panel_top_path = "{{colors.primary.default.hex}}"
    file_panel_item_selected_fg = "{{colors.on_surface.default.hex}}"
    file_panel_item_selected_bg = "{{colors.surface_container_lowest.default.hex}}"

    footer_fg = "{{colors.on_surface.default.hex}}"
    footer_bg = "{{colors.surface.default.hex}}"
    footer_border = "{{colors.outline.default.hex}}"
    footer_border_active = "{{colors.primary.default.hex}}"

    sidebar_fg = "{{colors.on_surface.default.hex}}"
    sidebar_bg = "{{colors.surface.default.hex}}"
    sidebar_title = "{{colors.secondary.default.hex}}"
    sidebar_border = "{{colors.outline.default.hex}}"
    sidebar_border_active = "{{colors.primary.default.hex}}"
    sidebar_item_selected_fg = "{{colors.on_surface.default.hex}}"
    sidebar_item_selected_bg = "{{colors.surface_container_lowest.default.hex}}"
    sidebar_divider = "{{colors.outline.default.hex}}"

    modal_fg = "{{colors.on_surface.default.hex}}"
    modal_bg = "{{colors.surface.default.hex}}"
    modal_border_active = "{{colors.primary.default.hex}}"
    modal_cancel_fg = "{{colors.on_tertiary.default.hex}}"
    modal_cancel_bg = "{{colors.tertiary.default.hex}}"
    modal_confirm_fg = "{{colors.on_primary.default.hex}}"
    modal_confirm_bg = "{{colors.primary.default.hex}}"

    help_menu_hotkey = "{{colors.primary.default.hex}}"
    help_menu_title = "{{colors.secondary.default.hex}}"

    cursor = "{{colors.primary.default.hex}}"
    correct = "{{colors.secondary.default.hex}}"
    error = "{{colors.error.default.hex}}"
    hint = "{{colors.tertiary.default.hex}}"
    cancel = "{{colors.error.default.hex}}"
  '';

  home.file.".config/noctalia/templates/kitty.conf".text = ''
    # Generated by Noctalia. Loaded by Kitty at window startup.
    foreground {{colors.on_surface.default.hex}}
    background {{colors.surface.default.hex}}
    cursor {{colors.primary.default.hex}}
    cursor_text_color {{colors.on_primary.default.hex}}
    selection_foreground {{colors.on_surface.default.hex}}
    selection_background {{colors.surface_container_lowest.default.hex}}
    active_border_color {{colors.primary.default.hex}}
    inactive_border_color {{colors.outline.default.hex}}
    active_tab_foreground {{colors.on_primary.default.hex}}
    active_tab_background {{colors.primary.default.hex}}
    inactive_tab_foreground {{colors.on_surface.default.hex}}
    inactive_tab_background {{colors.surface_container_lowest.default.hex}}
    color0 {{colors.surface.default.hex}}
    color1 {{colors.error.default.hex}}
    color2 {{colors.secondary.default.hex}}
    color3 {{colors.tertiary.default.hex}}
    color4 {{colors.primary.default.hex}}
    color5 {{colors.tertiary.default.hex}}
    color6 {{colors.secondary.default.hex}}
    color7 {{colors.on_surface.default.hex}}
    color8 {{colors.outline.default.hex}}
    color9 {{colors.error.default.hex}}
    color10 {{colors.secondary.default.hex}}
    color11 {{colors.tertiary.default.hex}}
    color12 {{colors.primary.default.hex}}
    color13 {{colors.tertiary.default.hex}}
    color14 {{colors.secondary.default.hex}}
    color15 {{colors.on_surface.default.hex}}
  '';

  # The Noctalia daemon replaces this fallback as soon as it starts.  Creating
  # it during Home Manager activation means Hyprland's `source` is valid even
  # on the first login.
  home.activation.gjallarNoctaliaHyprlandFallback = lib.hm.dag.entryAfter [ "writeBoundary" ] ''
    theme_file="$HOME/.local/state/noctalia/hyprland-colors.conf"
    if [ ! -e "$theme_file" ]; then
      mkdir -p "$(dirname "$theme_file")"
      cat >"$theme_file" <<'EOF'
$primary = rgb(7aa2f7)
$surface = rgb(1a1b26)
$secondary = rgb(7dcfff)
$error = rgb(f7768e)

general {
  col.active_border = $primary
  col.inactive_border = $surface
}

group {
  col.border_active = $secondary
  col.border_inactive = $surface
  col.border_locked_active = $error
  col.border_locked_inactive = $surface
}
EOF
    fi
  '';
}
