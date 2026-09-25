{
  config,
  inputs,
  lib,
  pkgs,
  ...
}:
let
  semanticTheme = import ../../../../themes/lib/semantic.nix { inherit config; };
  contrastGuard = import ../../../../themes/lib/contrast.nix { inherit pkgs; };

  # Reuse Stylix's widget artwork; Noctalia supplies the changing palette.
  roles = semanticTheme.kvantum.roles;
  keys = builtins.attrNames roles;
  tokens = map (key: "{{${key}-hex}}") keys;
  themeFiles = {
    kvconfig = builtins.readFile "${inputs.stylix}/modules/qt/kvconfig.mustache";
    svg = builtins.readFile "${inputs.stylix}/modules/qt/kvantum.svg.mustache";
  };
  renderTemplate = builtins.replaceStrings tokens (
    map (key: "{{colors.${roles.${key}}.default.hex_stripped}}") keys
  );
  renderFallback = builtins.replaceStrings tokens (
    map (key: builtins.getAttr key semanticTheme.kvantum.fallback) keys
  );
in
{
  qt = {
    enable = true;
    platformTheme.name = lib.mkForce "qtct";
    style.name = lib.mkForce "kvantum";
    kvantum.settings.General.theme = lib.mkForce "Noctalia";
  };

  programs.noctalia.settings.theme.templates.user = lib.mapAttrs' (
    extension: _:
    lib.nameValuePair "qt_kvantum_${extension}" {
      input_path = "$XDG_CONFIG_HOME/noctalia/templates/qt.${extension}";
      output_path = "$XDG_CONFIG_HOME/Kvantum/Noctalia/Noctalia.${extension}";
      pre_hook = contrastGuard.preHook;
    }
  ) themeFiles;

  xdg.configFile = lib.mapAttrs' (
    extension: content:
    lib.nameValuePair "noctalia/templates/qt.${extension}" {
      text = renderTemplate content;
    }
  ) themeFiles;

  # Writable outputs belong to Noctalia, with a first-login fallback palette.
  home.activation.noctaliaQtFallback = lib.hm.dag.entryAfter [ "writeBoundary" ] (
    lib.concatStringsSep "\n" (
      lib.mapAttrsToList (
        extension: content:
        let
          target = "${config.xdg.configHome}/Kvantum/Noctalia/Noctalia.${extension}";
          fallback = pkgs.writeText "noctalia-qt.${extension}" (renderFallback content);
        in
        ''
          if [ ! -e ${lib.escapeShellArg target} ]; then
            run ${pkgs.coreutils}/bin/install -D -m 0600 ${fallback} ${lib.escapeShellArg target}
          fi
        ''
      ) themeFiles
    )
  );
}
