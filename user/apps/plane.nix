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
  planeApplication = lib.findFirst (
    application: (application.id or "") == "plane"
  ) null webApplications;

  planeEnable =
    if canonicalWebApplications
    then planeApplication != null
    else settings.planeEnable or false;

  planeHost =
    if planeApplication != null
    then planeApplication.endpoint or ""
    else settings.planeHost or "";

  planeLauncher = webApplication.mkIsolatedWebApplication {
    name = "plane";
    browser = pkgs.brave;
    profile = ".config/gjallarOS/brave-plane";
    url = planeHost;
    browserArguments = [
      "--no-first-run"
      "--no-default-browser-check"
      "--disable-sync"
      "--disable-background-mode"
    ];
  };
in
{
  config = lib.mkIf planeEnable {
    home.packages = [
      planeLauncher
    ];



    xdg.desktopEntries.plane = {
      name = "Plane";
      comment = "Open the configured Plane workspace";
      exec = "${planeLauncher}/bin/plane";
      icon = "${pkgs.papirus-icon-theme}/share/icons/Papirus/64x64/apps/planner.svg";
      terminal = false;
      categories = [
        "Development"
        "Office"
        "ProjectManagement"
      ];
    };
  };
}
