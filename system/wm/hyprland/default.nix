{
  inputs,
  config,
  pkgs,
  lib,
  settings,
  ...
}:
let
  shell = settings.themeDetails.shell or "noctalia";
  hyprlandSession = pkgs.writeShellScriptBin "gjallar-hyprland-session" ''
    state_dir="''${XDG_STATE_HOME:-$HOME/.local/state}/hyprland"
    mkdir -p "$state_dir"
    export HYPRLAND_CONFIG="''${XDG_CONFIG_HOME:-$HOME/.config}/hypr/hyprland.conf"
    exec ${pkgs.hyprland}/bin/Hyprland >>"$state_dir/startup.log" 2>&1
  '';
  hyprlandSessionEntry = pkgs.runCommand "gjallar-hyprland-session-entry" { } ''
    install -Dm444 ${pkgs.writeText "gjallar-hyprland.desktop" ''
      [Desktop Entry]
      Name=GjallarOS Hyprland
      Comment=Hyprland session with quiet startup logging
      Exec=${hyprlandSession}/bin/gjallar-hyprland-session
      Type=Application
      DesktopNames=Hyprland
    ''} "$out/share/wayland-sessions/gjallar-hyprland.desktop"
  '';
in
{
  environment.systemPackages = [ hyprlandSession hyprlandSessionEntry ];

  imports = [
    ../common/wayland.nix
  ]
  ++ lib.optional (shell == "noctalia") ../shells/noctalia.nix;

  programs = {
    hyprland = {
      enable = true;
      xwayland.enable = true;
      # package = inputs.hyprland.packages.${pkgs.system}.default;
      # portalPackage = inputs.hyprland.packages.${pkgs.system}.xdg-desktop-portal-hyprland;
      package = pkgs.hyprland;
      portalPackage = pkgs.xdg-desktop-portal-hyprland;
    };
  };

  nix.settings = {
    substituters = [ "https://hyprland.cachix.org" ];
    trusted-public-keys = [ "hyprland.cachix.org-1:a7pgxzMz7+chwVL3/pzj6jIBMioiJM7ypFP8PwtkuGc=" ];
  };

  xdg.portal = {
    enable = true;
    xdgOpenUsePortal = true;
    config = {
      hyprland.default = [ "hyprland" ];
    };
  };
}
