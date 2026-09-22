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
  gjallarctl = pkgs.callPackage ../../../pkgs/gjallarctl { };
  moniquePackage = pkgs.callPackage ../../../pkgs/monique/nix/package.nix { };
  hyprlandSession = pkgs.writeShellScriptBin "gjallar-hyprland-session" ''
        state_dir="''${XDG_STATE_HOME:-$HOME/.local/state}/hyprland"
        noctalia_state_dir="''${XDG_STATE_HOME:-$HOME/.local/state}/noctalia"
        hypr_config_dir="''${XDG_CONFIG_HOME:-$HOME/.config}/hypr"
        mkdir -p "$state_dir"
        mkdir -p "$noctalia_state_dir" "$hypr_config_dir"

        if [ ! -e "$noctalia_state_dir/hyprland-colors.conf" ]; then
          cat >"$noctalia_state_dir/hyprland-colors.conf" <<'EOF'
    $primary = rgb(7aa2f7)
    $surface = rgb(1a1b26)
    $secondary = rgb(7dcfff)
    $error = rgb(f7768e)

    general {
      col.active_border = $primary
      col.inactive_border = $surface
    }

    group {
      col.border_active = $secondary
      col.border_inactive = $surface
      col.border_locked_active = $error
      col.border_locked_inactive = $surface
    }
    EOF
        fi
        for display_config in monitors.conf workspaces.conf; do
          [ -e "$hypr_config_dir/$display_config" ] || : >"$hypr_config_dir/$display_config"
        done

        export HYPRLAND_CONFIG="$hypr_config_dir/hyprland.conf"
        exec ${pkgs.hyprland}/bin/start-hyprland >>"$state_dir/startup.log" 2>&1
  '';
  hyprlandSessionEntry =
    pkgs.runCommand "gjallar-hyprland-session-entry"
      {
        passthru.providedSessions = [ "gjallar-hyprland" ];
      }
      ''
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
  environment.systemPackages = [ hyprlandSession ];

  services.displayManager.sessionPackages = [ hyprlandSessionEntry ];

  systemd.user.services.display-input-rotation =
    lib.mkIf (settings.orientationSensorEnable or false)
      {
        description = "Automatic display and input rotation";

        wantedBy = [ "graphical-session.target" ];
        after = [ "graphical-session.target" ];

        serviceConfig = {
          ExecStart = "${gjallarctl}/bin/gjallarctl hyprland-rotate";
          Restart = "on-failure";
          RestartSec = 1;
        };
      };

  imports = [
    ../common/wayland.nix
  ]
  ++ lib.optional (shell == "noctalia") ../shells/noctalia.nix;

  programs = {
    monique = {
      enable = true;
      package = moniquePackage;
      enablePolkit = false;
    };

    hyprland = {
      enable = true;
      xwayland.enable = true;
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
    extraPortals = [ pkgs.xdg-desktop-portal-gtk ];
    config.hyprland = {
      default = [
        "hyprland"
        "gtk"
      ];
      "org.freedesktop.impl.portal.AppChooser" = [ "gtk" ];
      "org.freedesktop.impl.portal.FileChooser" = [ "gtk" ];
    };
  };
}
