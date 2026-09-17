{
  inputs,
  config,
  lib,
  settings,
  pkgs,
  ...
}:
let
  # The selected theme owns the shell choice; keep Hyprland itself stable.
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
    ./scripts.nix
    ./rules.nix
    ./plugins.nix
    ./hyprlock.nix
    ../shells/${shell}
    ./settings.nix
  ];

  home.packages = with pkgs; [
    hyprcursor
  ];

  # Monique owns mutable display state.
  home.activation.gjallarMoniqueDisplayState = lib.hm.dag.entryAfter [ "writeBoundary" ] ''
          config_home="''${XDG_CONFIG_HOME:-$HOME/.config}"
          monique_dir="$config_home/monique"
          settings_file="$monique_dir/settings.json"
          hypr_dir="$config_home/hypr"

          ${pkgs.coreutils}/bin/mkdir -p "$monique_dir" "$hypr_dir"

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
    # package = inputs.hyprland.packages.${pkgs.system}.hyprland;
    package = pkgs.hyprland;
    systemd.enable = true;
    # Noctalia owns this generated include.  It is a normal user-writable
    # state file, while this Home Manager config remains immutable.
    extraConfig = ''
      source = ~/.local/state/noctalia/hyprland-colors.conf
      # Monique-generated display state.
      source = ~/.config/hypr/monitors.conf
    '';
    plugins = [
      # inputs.hypr-dynamic-cursors.packages.${pkgs.system}.hypr-dynamic-cursors
      # pkgs.hyprlandPlugins.hypr-dynamic-cursors
    ]
    ++
      lib.optional (settings.themeDetails.bordersPlusPlus)
        # inputs.hyprland-plugins.packages.${pkgs.system}.borders-plus-plus;
        pkgs.hyprlandPlugins.borders-plus-plus;
  };

  xdg.portal = {
    enable = true;
    xdgOpenUsePortal = true;
    config = {
      hyprland.default = [ "hyprland" ];
    };
  };
}
