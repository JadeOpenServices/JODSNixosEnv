{
  config,
  lib,
  pkgs,
  settings,
  ...
}:
let
  edge = "${pkgs.microsoft-edge}/bin/microsoft-edge";
  teamsProfile = "${config.home.homeDirectory}/.config/microsoft-edge-teams";
  normalBrowser = settings.preferredBrowser;

  teamsApp = pkgs.writeShellScriptBin "gjallar-teams" ''
    set -euo pipefail
    mkdir -p ${teamsProfile}
    if [ "$#" -gt 0 ]; then
        exec ${edge} --password-store=basic --class=gjallar-teams \
            --name="Microsoft Teams" --user-data-dir=${teamsProfile} --app="$1"
    fi
    exec ${edge} --password-store=basic --class=gjallar-teams \
        --name="Microsoft Teams" --user-data-dir=${teamsProfile} \
        --app=https://teams.microsoft.com
  '';

  urlHandler = pkgs.writeShellScriptBin "gjallar-open-url" ''
    set -euo pipefail
    url="''${1:-}"
    [ -n "$url" ] || exit 2
    case "$url" in
        https://teams.microsoft.com|https://teams.microsoft.com/*|https://teams.live.com|https://teams.live.com/*|https://teams.microsoft.us|https://teams.microsoft.us/*|https://teams.cloud.microsoft|https://teams.cloud.microsoft/*|msteams:*)
            exec ${teamsApp}/bin/gjallar-teams "$url"
            ;;
        *)
            exec ${normalBrowser} "$url"
            ;;
    esac
  '';
in
{
  home.packages = [
    pkgs.microsoft-edge
    teamsApp
    urlHandler
  ];

  # Keep the launcher icon independent from Edge's generic application icon.
  home.file.".local/share/icons/hicolor/scalable/apps/gjallar-teams.svg".source =
    "${pkgs.papirus-icon-theme}/share/icons/Papirus/64x64/apps/teams-for-linux.svg";

  xdg.mimeApps = {
    enable = true;
    defaultApplications = {
      "x-scheme-handler/http" = [ "gjallar-url-handler.desktop" ];
      "x-scheme-handler/https" = [ "gjallar-url-handler.desktop" ];
      "x-scheme-handler/msteams" = [ "gjallar-url-handler.desktop" ];
    };
  };

  home.file.".local/share/applications/gjallar-url-handler.desktop".text = ''
    [Desktop Entry]
    Type=Application
    Name=GjallarOS URL handler
    NoDisplay=true
    Exec=${urlHandler}/bin/gjallar-open-url %u
    MimeType=x-scheme-handler/http;x-scheme-handler/https;x-scheme-handler/msteams;
  '';

  home.file.".local/share/applications/microsoft-teams.desktop".text = ''
    [Desktop Entry]
    Type=Application
    Name=Microsoft Teams
    Exec=${teamsApp}/bin/gjallar-teams %U
    Terminal=false
    Icon=gjallar-teams
    Categories=Network;Office;InstantMessaging;
    StartupWMClass=gjallar-teams
  '';

  home.activation.gjallarDesktopDatabase = lib.hm.dag.entryAfter [ "writeBoundary" ] ''
    if command -v update-desktop-database >/dev/null 2>&1; then
        update-desktop-database "$HOME/.local/share/applications" >/dev/null 2>&1 || true
    fi
  '';
}
