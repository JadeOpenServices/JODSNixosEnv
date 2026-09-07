{
  config,
  lib,
  pkgs,
  settings,
  ...
}:
let
  drawioEnable = settings.drawioEnable or false;
  drawioSelfHosted = settings.drawioSelfHosted or false;
  drawioHost = settings.drawioHost or "";

  publicEndpoint = "https://app.diagrams.net/";
  endpoint =
    if drawioSelfHosted
    then drawioHost
    else publicEndpoint;

  appBrowser = lib.getExe pkgs.microsoft-edge;
  endpointFile = "${config.home.homeDirectory}/.config/gjallarOS/drawio-endpoint";

  drawioLauncher = pkgs.writeShellScriptBin "gjallar-drawio" ''
    set -euo pipefail

    endpoint_file=${lib.escapeShellArg endpointFile}

    if [ ! -r "$endpoint_file" ]; then
      printf 'Draw.io endpoint configuration is unavailable.\n' >&2
      false
    fi

    endpoint="$(cat "$endpoint_file")"

    if [ -z "$endpoint" ]; then
      printf 'Draw.io endpoint configuration is empty.\n' >&2
      false
    fi

    exec ${appBrowser}       --app="$endpoint"       --class=gjallar-drawio       --name=Draw.io
  '';
in
{
  config = lib.mkIf drawioEnable {
    home.packages = [
      drawioLauncher
    ];

    home.file.".config/gjallarOS/drawio-endpoint".text = endpoint;

    xdg.desktopEntries.drawio = {
      name = "Draw.io";
      genericName = "Diagram Editor";
      comment = "Open Draw.io";
      exec = "${drawioLauncher}/bin/gjallar-drawio";
      icon = "${pkgs.papirus-icon-theme}/share/icons/Papirus/64x64/apps/drawio.svg";
      terminal = false;
      categories = [
        "Development"
        "Graphics"
        "Office"
      ];
    };
  };
}
