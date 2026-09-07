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


  drawioLauncher = pkgs.writeShellScriptBin "gjallar-drawio" ''
    exec ${lib.getExe pkgs.brave} \
      --user-data-dir="$HOME/.config/gjallarOS/brave-drawio" \
      --app=${lib.escapeShellArg endpoint} \
      --no-first-run \
      --no-default-browser-check \
      --disable-sync \
      --disable-background-mode
  '';
in
{
  config = lib.mkIf drawioEnable {
    home.packages = [
      drawioLauncher
    ];
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
