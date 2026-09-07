{
  config,
  lib,
  pkgs,
  settings,
  ...
}:
let
  planeEnable = settings.planeEnable or false;
  planeHost = settings.planeHost or "";
  browser = lib.getExe pkgs.${settings.preferredBrowser};
  endpointFile = "${config.home.homeDirectory}/.config/gjallarOS/plane-endpoint";

  planeLauncher = pkgs.writeShellScriptBin "plane" ''
    set -euo pipefail

    endpoint_file=${lib.escapeShellArg endpointFile}

    if [ ! -r "$endpoint_file" ]; then
      printf 'Plane endpoint configuration is unavailable.\n' >&2
      exit 1
    fi

    endpoint="$(cat "$endpoint_file")"

    if [ -z "$endpoint" ]; then
      printf 'Plane endpoint configuration is empty.\n' >&2
      exit 1
    fi

    exec ${browser} "$endpoint"
  '';
in
{
  config = lib.mkIf planeEnable {
    home.packages = [
      planeLauncher
    ];

    home.file.".config/gjallarOS/plane-endpoint".text = planeHost;

    xdg.desktopEntries.plane = {
      name = "Plane";
      comment = "Open the configured Plane workspace";
      exec = "${planeLauncher}/bin/plane";
      terminal = false;
      categories = [
        "Network"
        "Office"
        "ProjectManagement"
      ];
    };
  };
}
