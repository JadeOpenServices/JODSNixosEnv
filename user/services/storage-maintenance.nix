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

in
{
  # Home Manager generations are GC roots and therefore need explicit
  # retention just like system generations.
  nix.gc = {
    automatic = true;
    dates = "weekly";
    persistent = true;
    randomizedDelaySec = "2h";
    options = "--delete-older-than 14d";
  };

  # Human-visible persistent container data lives here.
  #
  # GjallarOS-managed containers that need persistent user-accessible data
  # should bind mount subdirectories of this location rather than exposing
  # Podman's internal graph storage.
  home.sessionVariables = lib.mkIf settings.containersEnable {
    GJALLAR_CONTAINER_DATA_HOME = containerDataHome;
  };

  systemd.user.tmpfiles.rules =
    lib.optionals settings.containersEnable
      [
        "d %h/Containers 0700 - - -"
      ];

  home.packages =
    lib.optionals settings.containersEnable
      [
        containerStorageInfo
      ];

  # Rootless Podman storage belongs to the user. Automatically remove only
  # old images unused by every container. Containers, named volumes and
  # human-owned persistent data are deliberately preserved.
  systemd.user.services.gjallar-podman-storage-maintenance =
    lib.mkIf settings.containersEnable
      {
        Unit.Description = "GjallarOS rootless Podman storage maintenance";

        Service = {
          Type = "oneshot";
          ExecStart = "${podmanCleanup}/bin/gjallar-podman-storage-maintenance";
          Nice = 19;
          IOSchedulingClass = "idle";
        };
      };

  systemd.user.timers.gjallar-podman-storage-maintenance =
    lib.mkIf settings.containersEnable
      {
        Unit.Description = "GjallarOS rootless Podman storage maintenance timer";

        Timer = {
          OnCalendar = "weekly";
          Persistent = true;
          RandomizedDelaySec = "2h";
        };

        Install.WantedBy = [ "timers.target" ];
      };
}
