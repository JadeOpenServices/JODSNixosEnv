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
  semanticTheme = import ../../../../themes/lib/semantic.nix { inherit config; };
  contrastGuard = import ../../../../themes/lib/contrast.nix { inherit pkgs; };

  palette = {
    dark = {
      mPrimary = "#${semanticTheme.fallback.primary}";
      mOnPrimary = "#${semanticTheme.fallback.onPrimary}";
      mSecondary = "#${semanticTheme.fallback.secondary}";
      mOnSecondary = "#${semanticTheme.fallback.onSecondary}";
      mTertiary = "#${semanticTheme.fallback.tertiary}";
      mOnTertiary = "#${semanticTheme.fallback.onTertiary}";
      mError = "#${semanticTheme.fallback.error}";
      mOnError = "#${semanticTheme.fallback.onError}";
      mSurface = "#${semanticTheme.fallback.surface}";
      mOnSurface = "#${semanticTheme.fallback.onSurface}";
      mSurfaceVariant = "#${semanticTheme.fallback.surfaceVariant}";
      mOnSurfaceVariant = "#${semanticTheme.fallback.onSurfaceVariant}";
      mOutline = "#${semanticTheme.fallback.outline}";
      mShadow = "#${semanticTheme.fallback.shadow}";
      mHover = "#${semanticTheme.fallback.hover}";
      mOnHover = "#${semanticTheme.fallback.onHover}";

      terminal = {
        background = "#${semanticTheme.fallback.terminal.background}";
        foreground = "#${semanticTheme.fallback.terminal.foreground}";

        cursor = "#${semanticTheme.fallback.terminal.cursor}";
        cursorText = "#${semanticTheme.fallback.terminal.cursorText}";

        selectionBg = "#${semanticTheme.fallback.selection}";
        selectionFg = "#${semanticTheme.fallback.onSelection}";

        normal = {
          black = "#${semanticTheme.fallback.terminal.normal.black}";
          red = "#${semanticTheme.fallback.terminal.normal.red}";
          green = "#${semanticTheme.fallback.terminal.normal.green}";
          yellow = "#${semanticTheme.fallback.terminal.normal.yellow}";
          blue = "#${semanticTheme.fallback.terminal.normal.blue}";
          magenta = "#${semanticTheme.fallback.terminal.normal.magenta}";
          cyan = "#${semanticTheme.fallback.terminal.normal.cyan}";
          white = "#${semanticTheme.fallback.terminal.normal.white}";
        };

        bright = {
          black = "#${semanticTheme.fallback.terminal.bright.black}";
          red = "#${semanticTheme.fallback.terminal.bright.red}";
          green = "#${semanticTheme.fallback.terminal.bright.green}";
          yellow = "#${semanticTheme.fallback.terminal.bright.yellow}";
          blue = "#${semanticTheme.fallback.terminal.bright.blue}";
          magenta = "#${semanticTheme.fallback.terminal.bright.magenta}";
          cyan = "#${semanticTheme.fallback.terminal.bright.cyan}";
          white = "#${semanticTheme.fallback.terminal.bright.white}";
        };
      };
    };
  };
in
{
  imports = [
    ./gtk.nix
    ./qt.nix
  ];

  programs.noctalia = {
    enable = true;
    systemd.enable = false;
    package = pkgs.callPackage ../../../../pkgs/noctalia {
      noctalia = inputs.noctalia.packages.${pkgs.stdenv.hostPlatform.system}.default;
    };

    settings =
      let
        base = import ./settings.nix;
      in
      lib.recursiveUpdate base {

        theme = {
          mode = "dark";
          source = "builtin";
          builtin = "Noctalia";
          templates.user.hyprland = {
            input_path = "$XDG_CONFIG_HOME/noctalia/templates/hyprland.conf";
            output_path = "$XDG_STATE_HOME/noctalia/hyprland-colors.conf";
            post_hook = "${pkgs.hyprland}/bin/hyprctl reload; if ${lib.getExe config.programs.noctalia.package} msg status >/dev/null 2>&1; then ${lib.getExe config.programs.noctalia.package} msg greeter-sync >/dev/null 2>&1 || true; fi";
          };
          templates.user.ghostty = {
            input_path = "$XDG_CONFIG_HOME/noctalia/templates/ghostty.conf";
            output_path = "$XDG_CONFIG_HOME/ghostty/themes/noctalia";
      pre_hook = contrastGuard.preHook;
            post_hook = "${pkgs.systemd}/bin/systemctl reload --user app-com.mitchellh.ghostty.service >/dev/null 2>&1 || true";
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
          clipboard_enabled = false;
          corner_radius_scale = 2.0;
          font_family = themeDetails.font;
          polkit_agent = true;
          greeter_sync = {
            auto_sync = true;
            privilege_command = "/run/wrappers/bin/pkexec";
          };
          screenshot = {
            directory = "${config.home.homeDirectory}/Pictures/Screenshots";
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
            drawer = false;
            menu_on_left_click = [ "nextcloud" ];
            leading_separator = true;
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

          # System RAM. gpu_vram showed the APU's fixed 2 GiB carveout,
          # which sits near 88% and looked like a stuck RAM reading.
          ram = {
            type = "sysmon";
            stat = "ram_pct";
            scale = 0.98;
          };

          power_profile = {
            type = "power_profile";
            scale = 1.0;

            actions = {
              left = "power-cycle";

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

          divider_apps = {
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

            type = "taskbar";

            group_by_workspace = true;

            workspace_group_content = "icons";

            show_workspace_label = false;

            workspace_group_capsule = false;

            group_single_icon_per_app = true;

            only_active_workspace = false;

            hide_empty_workspaces = false;

            icon_scale = 1.20;

            scale = 1.0;

            show_active_indicator = true;

            active_opacity = 1.0;

            inactive_opacity = 0.88;

          };

          active_window = {

            type = "taskbar";

            only_active_workspace = true;

            show_all_outputs = false;

            group_by_workspace = false;

            show_window_title = true;

            window_title_max_width = 200;

            taskbar_max_width = 430;

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
            hide_when_no_media = true;
            type = "media";

            art_size = 16;

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
        weather.enabled = true;

        system.monitor = {
          enabled = true;

          cpu_poll_seconds = 2.0;

          gpu_poll_seconds = 5.0;

          memory_poll_seconds = 2.0;
          network_poll_seconds = 3.0;
          disk_poll_seconds = 10.0;
        };

        bar.default = {
          capsule_thickness = 0.78;
          font_scale = 1.0;
          widget_spacing = 5;
          scale = 1.06;
          thickness = 40;
          margin_edge = 0;
          margin_ends = 20;
          margin_opposite_edge = 0;

          padding = 14;

          radius = 12;
          radius_top_left = 20;
          radius_top_right = 20;

          radius_bottom_left = 12;
          radius_bottom_right = 12;
          concave_edge_corners = true;
          background_opacity = 0.94;
          shadow = false;

          start = [
            "active_window"
            "divider_left"
            "workspaces"
          ];
          center = [
            "cpu_usage"
            "cpu_temp"
            "ram"

            "divider_center_left"

            "center_clock"

            "divider_center_right"

            "network"
            "keyboard_layout"
            "notifications"
            "clipboard"
            "volume"
            "power_profile"
            "battery"
            "tray"
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
          directory = "${config.home.homeDirectory}/Pictures";
          automation.recursive = true;
          per_monitor_directories = false;
          fill_mode = "crop";
          transition_on_startup = false;
        }
        // lib.optionalAttrs (settings.backgroundNormal != "") {
          default.path = settings.backgroundNormal;
        };
      };
  };

  xdg.configFile."noctalia/palettes/stylix.json".text = builtins.toJSON palette;

  xdg.configFile."noctalia/templates/hyprland.conf".text = ''
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

  xdg.configFile."noctalia/templates/ghostty.conf".text = ''
    background = {{colors.surface.default.hex}}
    foreground = {{colors.on_surface.default.hex}}
    cursor-color = {{colors.primary.default.hex}}
    cursor-text = {{colors.on_primary.default.hex}}
    selection-background = {{colors.primary.default.hex}}
    selection-foreground = {{colors.on_primary.default.hex}}
    palette = 0={{colors.surface.default.hex}}
    palette = 1={{colors.error.default.hex}}
    palette = 2={{colors.secondary.default.hex}}
    palette = 3={{colors.tertiary.default.hex}}
    palette = 4={{colors.primary.default.hex}}
    palette = 5={{colors.tertiary.default.hex}}
    palette = 6={{colors.secondary.default.hex}}
    palette = 7={{colors.on_surface.default.hex}}
    palette = 8={{colors.outline.default.hex}}
    palette = 9={{colors.error.default.hex}}
    palette = 10={{colors.secondary.default.hex}}
    palette = 11={{colors.tertiary.default.hex}}
    palette = 12={{colors.primary.default.hex}}
    palette = 13={{colors.tertiary.default.hex}}
    palette = 14={{colors.secondary.default.hex}}
    palette = 15={{colors.on_surface.default.hex}}
  '';

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
