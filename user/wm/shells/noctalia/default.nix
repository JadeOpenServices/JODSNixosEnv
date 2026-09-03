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
  profileDetails = settings.profileDetails;
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
  monitorNames = lib.unique [
    profileDetails.monitorsPosition.left
    profileDetails.monitorsPosition.center
    profileDetails.monitorsPosition.right
  ];
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
        source = "custom";
        custom_palette = "stylix";
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
          lenght = 15;
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
        monitors = lib.genAttrs monitorNames (monitor: {
          # On single-panel laptops all three profile positions may
          # resolve to eDP-1; the unique list prevents duplicate
          # dynamic attributes while retaining the user wallpaper.
          path =
            if monitor == profileDetails.monitorsPosition.center then
              selectedBackground
            else
              wallpaperDetails.center;
        });
      };
    };
  };

  home.file.".config/noctalia/palettes/stylix.json".text = (builtins.toJSON palette);
}
