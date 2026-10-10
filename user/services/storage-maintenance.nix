{
  config,
  lib,
  pkgs,
  settings,
  ...
}:

let
  containerDataHome = "${config.home.homeDirectory}/Containers";

  podmanCleanup = pkgs.writeShellApplication {
    name = "gjallar-podman-storage-maintenance";

    runtimeInputs = with pkgs; [
      coreutils
      podman
    ];

    text = ''
      graphroot="$(podman info --format '{{.Store.GraphRoot}}')"

      case "$graphroot" in
        "$HOME"/*)
          ;;
        *)
          printf 'REFUSE: Podman graphroot is outside the user home: %s\n' \
            "$graphroot" >&2
          exit 1
          ;;
      esac

      podman image prune \
        --all \
        --force \
        --filter until=720h
    '';
  };

  containerStorageInfo = pkgs.writeShellApplication {
    name = "gjallar-container-storage";

    runtimeInputs = with pkgs; [
      coreutils
      podman
    ];

    text = ''
      graphroot="$(podman info --format '{{.Store.GraphRoot}}')"
      data_root="${containerDataHome}"

      case "''${1:-status}" in
        status)
          printf 'Podman engine storage:\n  %s\n' "$graphroot"
          du -sh "$graphroot" 2>/dev/null || true

          printf '\nUser container data:\n  %s\n' "$data_root"
          du -sh "$data_root" 2>/dev/null || true

          printf '\nPodman usage:\n'
          podman system df
          ;;

        detailed)
          printf 'Podman engine storage:\n  %s\n\n' "$graphroot"
          podman system df -v
          ;;

        volumes)
          printf 'Named Podman volumes:\n'
          podman volume ls
          ;;

        paths)
          printf 'engine=%s\n' "$graphroot"
          printf 'data=%s\n' "$data_root"
          ;;

        *)
          printf '%s\n' \
            'Usage: gjallar-container-storage {status|detailed|volumes|paths}' >&2
          exit 2
          ;;
      esac
    '';
  };

  containersEnable = builtins.elem "containers" settings.apps;
in
{
  home.sessionVariables = lib.mkIf containersEnable {
    GJALLAR_CONTAINER_DATA_HOME = containerDataHome;
  };

  systemd.user.tmpfiles.rules =
    lib.optionals containersEnable
      [
        "d %h/Containers 0700 - - -"
      ];

  home.packages =
    lib.optionals containersEnable
      [
        containerStorageInfo
        (pkgs.writeTextDir "share/zsh/site-functions/_gjallar-container-storage" ''
          #compdef gjallar-container-storage
          _arguments '1:view:(status detailed volumes paths)'
        '')
      ];

  systemd.user.services.podman-storage-maintenance =
    lib.mkIf containersEnable
      {
        Unit.Description = "Rootless Podman storage maintenance";

        Service = {
          Type = "oneshot";
          ExecStart = "${podmanCleanup}/bin/gjallar-podman-storage-maintenance";
          Nice = 19;
          IOSchedulingClass = "idle";
        };
      };

  systemd.user.timers.podman-storage-maintenance =
    lib.mkIf containersEnable
      {
        Unit.Description = "Rootless Podman storage maintenance timer";

        Timer = {
          OnCalendar = "weekly";
          Persistent = true;
          RandomizedDelaySec = "2h";
        };

        Install.WantedBy = [ "timers.target" ];
      };
}
