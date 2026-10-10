{
  inputs,
  config,
  lib,
  settings,
  pkgs,
  ...
}:
let
  shell = settings.themeDetails.shell or "noctalia";

in
{
  _module.args.hyprlandShellDetails = import (../. + "/shells/${shell}/details.nix") {
    inherit
      pkgs
      inputs
      lib
      config
      ;
  };
  imports = [
    ./env.nix
    ./binds.nix
    ./rules.nix
    ./plugins.nix
    ./touch.nix
    ./hyprlock.nix
    ../shells/${shell}
    ./settings.nix
  ];

  home.packages = with pkgs; [
    hyprcursor
  ];

  home.activation.gjallarMoniqueDisplayState = lib.hm.dag.entryAfter [ "writeBoundary" ] ''
          config_home="''${XDG_CONFIG_HOME:-$HOME/.config}"
          monique_dir="$config_home/monique"
          settings_file="$monique_dir/settings.json"
          hypr_dir="$config_home/hypr"

          ${pkgs.coreutils}/bin/mkdir -p "$monique_dir" "$hypr_dir"

          if [ ! -f "$settings_file" ] \
            || ! ${pkgs.gnugrep}/bin/grep -Eq '"hypr_config_format"[[:space:]]*:[[:space:]]*"legacy"' "$settings_file" \
            || ! ${pkgs.gnugrep}/bin/grep -Eq '"monitor_config_name"[[:space:]]*:[[:space:]]*"monitors"' "$settings_file"
          then
            ${pkgs.python3}/bin/python3 - "$settings_file" <<'PY'
    import json
    import os
    import sys

    path = sys.argv[1]

    try:
        with open(path, encoding="utf-8") as handle:
            settings = json.load(handle)
    except (FileNotFoundError, json.JSONDecodeError):
        settings = {}

    if not isinstance(settings, dict):
        settings = {}

    settings["hypr_config_format"] = "legacy"
    settings["monitor_config_name"] = "monitors"

    tmp = path + ".tmp"
    with open(tmp, "w", encoding="utf-8") as handle:
        json.dump(settings, handle, indent=2, sort_keys=True)
        handle.write("\n")

    os.chmod(tmp, 0o600)
    os.replace(tmp, path)
    PY
          fi

          ${pkgs.coreutils}/bin/rm -f \
            "$hypr_dir/monitors.lua" \
            "$hypr_dir/workspaces.conf" \
            "$hypr_dir/workspaces.lua"

          if [ ! -e "$hypr_dir/monitors.conf" ]; then
            : > "$hypr_dir/monitors.conf"
          fi
  '';

  wayland.windowManager.hyprland = {
    enable = true;
    configType = "hyprlang";
    package = pkgs.hyprland;
    # Portals come from the system module (hyprland + gtk). The HM default
    # points the portal at a per-user dir with only hyprland.portal, which
    # drops AppChooser and with it OpenURI, so xdg-open fails.
    portalPackage = null;
    systemd.enable = true;
    # Noctalia and the apps it launches run as user units. They need the
    # whole session environment, XDG_SESSION_ID for the polkit agent among it.
    systemd.variables = [ "--all" ];
    extraConfig = ''
      source = ~/.local/state/noctalia/hyprland-colors.conf
      source = ~/.config/hypr/monitors.conf
    '';
  };

}
