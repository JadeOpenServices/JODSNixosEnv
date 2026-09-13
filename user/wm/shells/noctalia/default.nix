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

        # Visible themed selection block.
        selectionBg = "#${config.lib.stylix.colors.base0D}";
        selectionFg = "#${config.lib.stylix.colors.base00}";

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
    systemd.enable = false;
    package = inputs.noctalia.packages.${pkgs.stdenv.hostPlatform.system}.default;

    settings = lib.recursiveUpdate (import ./settings.nix) {

      theme = {
        mode = "dark";
        source = "builtin";
        builtin = "Noctalia";
        templates.user.hyprland = {
          input_path = "$XDG_CONFIG_HOME/noctalia/templates/hyprland.conf";
          output_path = "$XDG_STATE_HOME/noctalia/hyprland-colors.conf";
          post_hook = "${pkgs.hyprland}/bin/hyprctl reload; if ${lib.getExe config.programs.noctalia.package} msg status >/dev/null 2>&1; then ${lib.getExe config.programs.noctalia.package} msg greeter-sync >/dev/null 2>&1 || true; fi";
        };
        templates.user.yazi = {
          input_path = "$XDG_CONFIG_HOME/noctalia/templates/yazi.toml";
          output_path = "$XDG_CONFIG_HOME/yazi/theme.toml";
        };
        templates.user.yazi_disk = {
          input_path = "$XDG_CONFIG_HOME/noctalia/templates/yazi-init.lua";
          output_path = "$XDG_CONFIG_HOME/yazi/init.lua";
        };
        templates.user.kitty = {
          input_path = "$XDG_CONFIG_HOME/noctalia/templates/kitty.conf";
          output_path = "$XDG_CONFIG_HOME/kitty/noctalia.conf";
        };
        templates.user.gjallar_plymouth = {
          input_path = "$XDG_CONFIG_HOME/noctalia/templates/stylix-override.json";
          output_path = "$XDG_STATE_HOME/noctalia/stylix-override.json";
        };
      };
      accessibility = {
        ui_scale = 1.4;
      };
      shell = {
        # Privacy: Noctalia must not persist clipboard contents.
        clipboard_enabled = false;
        corner_radius_scale = 2.0;
        font_family = themeDetails.font;
        avatar_path = avatarPath;
        polkit_agent = true;
        greeter_sync = {
          auto_sync = true;
          privilege_command = "/run/wrappers/bin/pkexec";
        };
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
          scale = 0.98;
          show_glyph = false;
        };
        volume = {
          scale = 0.98;
          show_label = false;
        };
        spacer_default = {
          type = "spacer";
          length = 15;
        };
        audio_visualizer = {
          width = 120;
        };
        tray = {
          drawer = true;
        };
        cpu_usage = {
          type = "sysmon";
          stat = "cpu_usage";
          scale = 0.98;
        };

        cpu_temp = {
          type = "sysmon";
          stat = "cpu_temp";
          scale = 0.98;
        };

        gpu_vram = {
          type = "sysmon";
          stat = "gpu_vram";
          scale = 0.98;
        };

        power_profile = {
          type = "power_profile";
          scale = 1.0;

          actions = {
            # Fast profile switching directly from the bar.
            left = "power-cycle";

            # Explicit profile selection / full power settings.
            right = "panel-toggle control-center power";
          };

        };

        battery = {
          type = "battery";
          display_mode = "glyph";
          show_label = true;
          label_content = "percent";
          scale = 0.98;
        };

        divider_left = {
          type = "text";
          text = "│";
          scale = 0.72;
          font_scale = 0.9;
          interactive = false;
        };

        divider_diag = {
          type = "text";
          text = "│";
          scale = 0.72;
          font_scale = 0.9;
          interactive = false;
        };

        divider_media = {
          type = "text";
          text = "│";
          scale = 0.72;
          font_scale = 0.9;
          interactive = false;
        };

        workspaces = {

          # Native workspace-aware application indicators.

          # Occupied workspaces show their application icons instead of dots.

          type = "taskbar";

          group_by_workspace = true;

          workspace_group_content = "icons";

          # Keep the bar compact: the icons themselves represent occupied

          # workspaces, without the numbered workspace badge.

          show_workspace_label = false;

          workspace_group_capsule = false;

          # Collapse multiple windows from the same application.

          group_single_icon_per_app = true;

          # Show all persistent workspaces rather than only the current one.

          only_active_workspace = false;

          hide_empty_workspaces = false;

          # Slightly larger, readable icons without making the 40px bar huge.

          icon_scale = 1.20;

          scale = 1.0;

          show_active_indicator = true;

          active_opacity = 1.0;

          inactive_opacity = 0.88;

        };

        active_window = {

          # Use Noctalia's supported taskbar title rendering instead of the

          # active_window widget, whose title width is not configurable in v5.

          type = "taskbar";

          # Behave like an active-workspace application title strip.

          only_active_workspace = true;

          show_all_outputs = false;

          group_by_workspace = false;

          # Show the application icon and its window title.

          show_window_title = true;

          # Enough room for normal application names/titles without allowing

          # pathological browser/document titles to invade the centre widgets.

          window_title_max_width = 360;

          taskbar_max_width = 430;

          # Slightly larger, readable app icon.

          icon_scale = 1.18;

          item_spacing = 5;

          show_active_indicator = false;

          active_opacity = 1.0;

          inactive_opacity = 0.92;

          scale = 1.0;

          font_scale = 1.0;

        };

        weather = {
          type = "weather";
          scale = 0.98;
        };

        media = {
          type = "media";

          # Keep the far-right media widget compact.
          art_size = 16;
          hide_when_no_media = false;

          scale = 0.98;
        };

        session = {
          type = "session";
          scale = 0.98;

          actions = {
            left = "panel-toggle session";
          };
        };

        divider_center_left = {
          type = "text";
          text = "│";
          scale = 0.72;
          font_scale = 0.9;
          interactive = false;
        };

        divider_center_right = {
          type = "text";
          text = "│";
          scale = 0.72;
          font_scale = 0.9;
          interactive = false;
        };

        center_clock = {
          type = "clock";
          format = "{:%a %d %b  %H:%M}";
          scale = 0.97;
          font_scale = 1.0;
          anchor = true;
        };

      };
      location = {
        auto_locate = false;
        address = "${settings.weatherCity}, ${settings.weatherCountry}";
      };
      osd = {
        orientation = "horizontal";
        position = "top_center";
        kinds = {
          keyboard_layout = false;
        };
      };
      # gjallarOS Noctalia bar services BEGIN
      weather.enabled = true;

      system.monitor = {
        enabled = true;

        # CPU usage + Tctl temperature.
        cpu_poll_seconds = 2.0;

        # GPU usage/temperature/VRAM. Noctalia only actively probes
        # GPU data while a GPU sysmon stat is actually displayed.
        gpu_poll_seconds = 5.0;

        memory_poll_seconds = 2.0;
        network_poll_seconds = 3.0;
        disk_poll_seconds = 10.0;
      };
      # gjallarOS Noctalia bar services END

      bar.default = {
        capsule_thickness = 0.78;
        font_scale = 1.0;
        widget_spacing = 5;
        scale = 1.06;
        thickness = 40;
        margin_edge = 0;
        margin_ends = 20;
        margin_opposite_edge = 0;

        # Remove the old outer content padding.
        padding = 14;

        # Top corners touch the physical screen corners.
        radius = 12;
        radius_top_left = 20;
        radius_top_right = 20;

        # Smooth inner wave.
        radius_bottom_left = 12;
        radius_bottom_right = 12;
        concave_edge_corners = true;
        background_opacity = 0.94;
        shadow = false;

        # Catppuccin-like card/capsule feel, but colors still come from
        # Noctalia's current palette/theme.
        start = [
          "active_window"
          "divider_left"
          "workspaces"
        ];
        center = [
          "cpu_usage"
          "cpu_temp"
          "gpu_vram"

          "divider_center_left"

          "center_clock"

          "divider_center_right"

          "network"
          "tray"
          "keyboard_layout"
          "notifications"
          "clipboard"
          "volume"
          "power_profile"
          "battery"
        ];
        end = [
          "weather"
          "divider_media"
          "media"
          "divider_power"
          "session"
        ];
      };
      wallpaper = {
        # Keep the picker on the user's real wallpaper library. The configured
        # XDG pictures directory is ~/Media/Pictures, but downloaded/user
        # wallpapers are intentionally kept in ~/Pictures.
        directory = "${config.home.homeDirectory}/Pictures";
        # Include image folders added below ~/Pictures as well as images in its
        # root.  The picker still allows normal folder navigation.
        automation.recursive = true;
        per_monitor_directories = false;
        fill_mode = "crop";
        transition_on_startup = false;
        # This is only the initial image.  The picker persists later choices
        # in Noctalia's state file without narrowing the browse directory.
        default.path = selectedBackground;
      };
    };
  };

  xdg.configFile."noctalia/palettes/stylix.json".text = builtins.toJSON palette;

  xdg.configFile."noctalia/templates/hyprland.conf".text = ''
    # Generated by Noctalia and reloaded by the template post-hook.
    $primary = rgb({{colors.primary.default.hex_stripped}})
    $surface = rgb({{colors.surface.default.hex_stripped}})
    $secondary = rgb({{colors.secondary.default.hex_stripped}})
    $error = rgb({{colors.error.default.hex_stripped}})

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
  '';

  xdg.configFile."noctalia/templates/kitty.conf".text = ''
    # Generated by Noctalia. Loaded by Kitty at window startup.
    foreground {{colors.on_surface.default.hex}}
    background {{colors.surface.default.hex}}
    cursor {{colors.primary.default.hex}}
    cursor_text_color {{colors.on_primary.default.hex}}
    # Windows-style visible selection block using the active Noctalia accent.
    selection_foreground {{colors.on_primary.default.hex}}
    selection_background {{colors.primary.default.hex}}
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

  # Plymouth lives in the initrd, so it cannot consume Noctalia's runtime
  # palette directly.  This bridge snapshots the active palette; `rebuild`
  # passes it into Stylix while producing the next boot generation.
  xdg.configFile."noctalia/templates/stylix-override.json".text = ''
    {
      "base00": "{{colors.surface.default.hex_stripped}}",
      "base01": "{{colors.surface_variant.default.hex_stripped}}",
      "base02": "{{colors.surface_container_lowest.default.hex_stripped}}",
      "base03": "{{colors.outline.default.hex_stripped}}",
      "base04": "{{colors.outline.default.hex_stripped}}",
      "base05": "{{colors.on_surface.default.hex_stripped}}",
      "base06": "{{colors.on_surface.default.hex_stripped}}",
      "base07": "{{colors.on_surface.default.hex_stripped}}",
      "base08": "{{colors.error.default.hex_stripped}}",
      "base09": "{{colors.tertiary.default.hex_stripped}}",
      "base0A": "{{colors.secondary.default.hex_stripped}}",
      "base0B": "{{colors.secondary.default.hex_stripped}}",
      "base0C": "{{colors.tertiary.default.hex_stripped}}",
      "base0D": "{{colors.primary.default.hex_stripped}}",
      "base0E": "{{colors.tertiary.default.hex_stripped}}",
      "base0F": "{{colors.error.default.hex_stripped}}"
    }
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
