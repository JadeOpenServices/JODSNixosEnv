{
  config,
  lib,
  pkgs,
  settings,
  ...
}:
let
  webApplication = import ./lib/web-application.nix { inherit lib pkgs; };

  canonicalWebApplications = settings ? webApplications;
  webApplications = settings.webApplications or [ ];
  drawioApplication = lib.findFirst (
    application: (application.id or "") == "drawio"
  ) null webApplications;

  drawioEnable =
    if canonicalWebApplications
    then drawioApplication != null
    else settings.drawioEnable or false;

  drawioHost =
    if drawioApplication != null
    then drawioApplication.endpoint or ""
    else settings.drawioHost or "";

  drawioSelfHosted =
    if canonicalWebApplications
    then drawioEnable && drawioHost != ""
    else settings.drawioSelfHosted or false;

  publicEndpoint = "https://app.diagrams.net/";
  endpoint =
    if drawioSelfHosted
    then drawioHost
    else publicEndpoint;

  drawioLauncher = webApplication.mkIsolatedWebApplication {
    name = "gjallar-drawio";
    browser = pkgs.brave;
    profile = ".config/gjallarOS/brave-drawio";
    url = endpoint;
    browserArguments = [
      "--no-first-run"
      "--no-default-browser-check"
      "--disable-sync"
      "--disable-background-mode"
    ];
  };
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
