{
  settings,
  config,
  lib,
  pkgs,
  ...
}:
let
  semanticTheme = import ../../../themes/lib/semantic.nix { inherit config; };
  contrastGuard = import ../../../themes/lib/contrast.nix { inherit pkgs; };

  hyprlockTemplate = ''
    $gjallar_surface = rgb({{colors.surface.default.hex_stripped}})
    $gjallar_on_surface = rgb({{colors.on_surface.default.hex_stripped}})
    $gjallar_primary = rgb({{colors.primary.default.hex_stripped}})
    $gjallar_on_primary = rgb({{colors.on_primary.default.hex_stripped}})
    $gjallar_error = rgb({{colors.error.default.hex_stripped}})
  '';

  hyprlockFallback = pkgs.writeText "gjallar-hyprlock-colors.conf" ''
    $gjallar_surface = rgb(${semanticTheme.fallback.surface})
    $gjallar_on_surface = rgb(${semanticTheme.fallback.onSurface})
    $gjallar_primary = rgb(${semanticTheme.fallback.primary})
    $gjallar_on_primary = rgb(${semanticTheme.fallback.onPrimary})
    $gjallar_error = rgb(${semanticTheme.fallback.error})
  '';
in
{
  programs.hyprlock.enable = true;
  programs.hyprlock.sourceFirst = true;

  programs.hyprlock.settings = {
    source = "${config.xdg.stateHome}/noctalia/hyprlock-colors.conf";
    path = "screenshot";
    general = {
      grace = 0;
      ignore_empty_input = true;
    };

    background = {
      path = "screenshot";
      blur_passes = 3;
      blur_size = 10;
      brightness = 1.0;
      contrast = 1.0;
      noise = 0.02;
    };

    input-field = {
      monitor = "";
      size = "250, 50";
      outline_thickness = 0;
      dots_size = 0.26;
      inner_color = "$gjallar_on_surface";
      dots_spacing = 0.64;
      dots_center = true;
      fade_on_empty = true;
      placeholder_text = "<i>Password...</i>";
      hide_input = false;
      check_color = "$gjallar_primary";
      position = "0, 50";
      halign = "center";
      valign = "bottom";
    };
  };
  programs.hyprlock.extraConfig = ''
    label {
        monitor =
        text = cmd[update:1000] echo "<b><big> $(date +"%H:%M") </big></b>"
        color = "$gjallar_on_surface";

        font_size = 64
        font_family = JetBrains Mono Nerd Font 10

        position = 0, -70
        halign = center
        valign = center
    }

    label {
        monitor =
        text = cmd[update:18000000] echo "<b> "$(date +'%A, %-d %B %Y')" </b>"
        color = "$gjallar_on_surface";

        font_size = 24
        font_family = JetBrains Mono Nerd Font 10

        position = 0, -120
        halign = center
        valign = center
    }
  '';

  programs.noctalia.settings.theme.templates.user.hyprlock = {
    input_path = "$XDG_CONFIG_HOME/noctalia/templates/hyprlock.conf";
    output_path = "$XDG_STATE_HOME/noctalia/hyprlock-colors.conf";
    pre_hook = contrastGuard.preHook;
  };

  xdg.configFile."noctalia/templates/hyprlock.conf".text = hyprlockTemplate;

  home.activation.noctaliaHyprlockFallback =
    lib.hm.dag.entryAfter [ "writeBoundary" ] ''
      target="${config.xdg.stateHome}/noctalia/hyprlock-colors.conf"

      if [ ! -e "$target" ]; then
        run ${pkgs.coreutils}/bin/install \
          -D -m 0600 \
          ${hyprlockFallback} \
          "$target"
      fi
    '';
}
