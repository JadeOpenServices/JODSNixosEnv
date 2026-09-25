{
  config,
  lib,
  pkgs,
  ...
}:
let
  semanticTheme = import ../../../themes/lib/semantic.nix { inherit config; };
  contrastGuard = import ../../../themes/lib/contrast.nix { inherit pkgs; };

  btopTemplate = ''
    theme[main_bg]="{{colors.surface.default.hex}}"
    theme[main_fg]="{{colors.on_surface.default.hex}}"
    theme[title]="{{colors.on_surface.default.hex}}"
    theme[hi_fg]="{{colors.primary.default.hex}}"

    theme[selected_bg]="{{colors.primary.default.hex}}"
    theme[selected_fg]="{{colors.on_primary.default.hex}}"

    theme[inactive_fg]="{{colors.outline.default.hex}}"
    theme[graph_text]="{{colors.on_surface.default.hex}}"
    theme[meter_bg]="{{colors.outline.default.hex}}"
    theme[proc_misc]="{{colors.on_surface.default.hex}}"

    theme[cpu_box]="{{colors.tertiary.default.hex}}"
    theme[mem_box]="{{colors.secondary.default.hex}}"
    theme[net_box]="{{colors.tertiary.default.hex}}"
    theme[proc_box]="{{colors.primary.default.hex}}"

    theme[div_line]="{{colors.surface_variant.default.hex}}"

    theme[temp_start]="{{colors.secondary.default.hex}}"
    theme[temp_mid]="{{colors.secondary.default.hex}}"
    theme[temp_end]="{{colors.error.default.hex}}"

    theme[cpu_start]="{{colors.secondary.default.hex}}"
    theme[cpu_mid]="{{colors.secondary.default.hex}}"
    theme[cpu_end]="{{colors.error.default.hex}}"

    theme[free_start]="{{colors.secondary.default.hex}}"
    theme[free_mid]="{{colors.secondary.default.hex}}"
    theme[free_end]="{{colors.secondary.default.hex}}"

    theme[cached_start]="{{colors.tertiary.default.hex}}"
    theme[cached_mid]="{{colors.tertiary.default.hex}}"
    theme[cached_end]="{{colors.secondary.default.hex}}"

    theme[available_start]="{{colors.error.default.hex}}"
    theme[available_mid]="{{colors.secondary.default.hex}}"
    theme[available_end]="{{colors.secondary.default.hex}}"

    theme[used_start]="{{colors.secondary.default.hex}}"
    theme[used_mid]="{{colors.tertiary.default.hex}}"
    theme[used_end]="{{colors.error.default.hex}}"

    theme[download_start]="{{colors.secondary.default.hex}}"
    theme[download_mid]="{{colors.secondary.default.hex}}"
    theme[download_end]="{{colors.error.default.hex}}"

    theme[upload_start]="{{colors.secondary.default.hex}}"
    theme[upload_mid]="{{colors.secondary.default.hex}}"
    theme[upload_end]="{{colors.error.default.hex}}"

    theme[process_start]="{{colors.secondary.default.hex}}"
    theme[process_mid]="{{colors.secondary.default.hex}}"
    theme[process_end]="{{colors.error.default.hex}}"
  '';

  btopFallback = pkgs.writeText "gjallar-btop.theme" ''
    theme[main_bg]="#${semanticTheme.fallback.surface}"
    theme[main_fg]="#${semanticTheme.fallback.onSurface}"
    theme[title]="#${semanticTheme.fallback.onSurface}"
    theme[hi_fg]="#${semanticTheme.fallback.primary}"

    theme[selected_bg]="#${semanticTheme.fallback.selection}"
    theme[selected_fg]="#${semanticTheme.fallback.onSelection}"

    theme[inactive_fg]="#${semanticTheme.fallback.outline}"
    theme[graph_text]="#${semanticTheme.fallback.onSurface}"
    theme[meter_bg]="#${semanticTheme.fallback.outline}"
    theme[proc_misc]="#${semanticTheme.fallback.onSurface}"

    theme[cpu_box]="#${semanticTheme.fallback.tertiary}"
    theme[mem_box]="#${semanticTheme.fallback.secondary}"
    theme[net_box]="#${semanticTheme.fallback.tertiary}"
    theme[proc_box]="#${semanticTheme.fallback.primary}"

    theme[div_line]="#${semanticTheme.fallback.surfaceVariant}"

    theme[temp_start]="#${semanticTheme.fallback.secondary}"
    theme[temp_mid]="#${semanticTheme.fallback.secondary}"
    theme[temp_end]="#${semanticTheme.fallback.error}"

    theme[cpu_start]="#${semanticTheme.fallback.secondary}"
    theme[cpu_mid]="#${semanticTheme.fallback.secondary}"
    theme[cpu_end]="#${semanticTheme.fallback.error}"

    theme[free_start]="#${semanticTheme.fallback.secondary}"
    theme[free_mid]="#${semanticTheme.fallback.secondary}"
    theme[free_end]="#${semanticTheme.fallback.secondary}"

    theme[cached_start]="#${semanticTheme.fallback.tertiary}"
    theme[cached_mid]="#${semanticTheme.fallback.tertiary}"
    theme[cached_end]="#${semanticTheme.fallback.secondary}"

    theme[available_start]="#${semanticTheme.fallback.error}"
    theme[available_mid]="#${semanticTheme.fallback.secondary}"
    theme[available_end]="#${semanticTheme.fallback.secondary}"

    theme[used_start]="#${semanticTheme.fallback.secondary}"
    theme[used_mid]="#${semanticTheme.fallback.tertiary}"
    theme[used_end]="#${semanticTheme.fallback.error}"

    theme[download_start]="#${semanticTheme.fallback.secondary}"
    theme[download_mid]="#${semanticTheme.fallback.secondary}"
    theme[download_end]="#${semanticTheme.fallback.error}"

    theme[upload_start]="#${semanticTheme.fallback.secondary}"
    theme[upload_mid]="#${semanticTheme.fallback.secondary}"
    theme[upload_end]="#${semanticTheme.fallback.error}"

    theme[process_start]="#${semanticTheme.fallback.secondary}"
    theme[process_mid]="#${semanticTheme.fallback.secondary}"
    theme[process_end]="#${semanticTheme.fallback.error}"
  '';
in
{
  programs.btop = {
    enable = true;
    package = pkgs.btop.override { rocmSupport = true; };
    settings = {
      color_theme = "gjallar";
      theme_background = false;
      vim_keys = true;
      update_ms = 500;
    };
  };

  programs.noctalia.settings.theme.templates.user.btop = {
    input_path = "$XDG_CONFIG_HOME/noctalia/templates/btop.theme";
    output_path = "$XDG_CONFIG_HOME/btop/themes/gjallar.theme";
    pre_hook = contrastGuard.preHook;
    post_hook = "${pkgs.procps}/bin/pkill -USR2 -x btop >/dev/null 2>&1 || true";
  };

  xdg.configFile."noctalia/templates/btop.theme".text = btopTemplate;

  home.activation.noctaliaBtopFallback =
    lib.hm.dag.entryAfter [ "writeBoundary" ] ''
      target="${config.xdg.configHome}/btop/themes/gjallar.theme"

      if [ ! -e "$target" ]; then
        run ${pkgs.coreutils}/bin/install \
          -D -m 0600 \
          ${btopFallback} \
          "$target"
      fi
    '';
}
