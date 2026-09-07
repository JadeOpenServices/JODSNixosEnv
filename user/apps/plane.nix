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

  planeLauncher = pkgs.writeShellScriptBin "plane" ''
    exec ${lib.getExe pkgs.brave} \
      --user-data-dir="$HOME/.config/gjallarOS/brave-plane" \
      --app=${lib.escapeShellArg planeHost} \
      --no-first-run \
      --no-default-browser-check \
      --disable-sync \
      --disable-background-mode
  '';
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
