{ pkgs, ... }:

let
  backgroundWeight = 80;
  foregroundWeight = 100;

  gjallarRun = pkgs.writeShellApplication {
    name = "gjallar-run";

    runtimeInputs = [
      pkgs.app2unit
    ];

    text = ''
      if [ "$#" -eq 0 ]; then
        echo "usage: gjallar-run COMMAND [ARG...]" >&2
        exit 64
      fi

      command_name="''${1##*/}"

      exec app2unit \
        -a "gjallar-$command_name" \
        -p CPUWeight=${toString backgroundWeight} \
        -- "$@"
    '';
  };

  gjallarResourceQos = pkgs.writeShellApplication {
    name = "gjallar-resource-qos";

    runtimeInputs = with pkgs; [
      coreutils
      gnused
      hyprland
      socat
      systemd
    ];

    text = ''
      background_weight=${toString backgroundWeight}
      foreground_weight=${toString foregroundWeight}
      current_unit=""

      set_weight() {
        unit="$1"
        weight="$2"

        [ -n "$unit" ] || return 0

        # The app may disappear between a focus event and this update.
        systemctl --user set-property \
          --runtime \
          "$unit" \
          "CPUWeight=$weight" \
          >/dev/null 2>&1 || true
      }

      focused_owned_unit() {
        active="$(
          hyprctl activewindow -j 2>/dev/null || true
        )"

        pid="$(
          printf '%s\n' "$active" |
            sed -n \
              's/^[[:space:]]*"pid":[[:space:]]*\([0-9][0-9]*\),\{0,1\}$/\1/p' |
            head -n 1
        )"

        [ -n "$pid" ] || return 1
        [ -r "/proc/$pid/cgroup" ] || return 1

        cgroup="$(
          sed -n 's#^0::##p' "/proc/$pid/cgroup"
        )"

        unit="''${cgroup##*/}"

        # app2unit escapes '-' inside its application-name component.
        # Match only workloads explicitly tagged by gjallar-run.
        case "$unit" in
          app-*-gjallar\\x2d*.scope|app-*-gjallar-*.scope)
            printf '%s\n' "$unit"
            ;;
          *)
            return 1
            ;;
        esac
      }

      update_focus() {
        next_unit="$(focused_owned_unit || true)"

        [ "$next_unit" = "$current_unit" ] && return 0

        if [ -n "$current_unit" ]; then
          set_weight "$current_unit" "$background_weight"
        fi

        if [ -n "$next_unit" ]; then
          set_weight "$next_unit" "$foreground_weight"
        fi

        current_unit="$next_unit"
      }

      runtime_dir="''${XDG_RUNTIME_DIR:-/run/user/$(id -u)}"

      if [ -z "''${HYPRLAND_INSTANCE_SIGNATURE:-}" ]; then
        printf '%s\n' \
          'gjallar-resource-qos: HYPRLAND_INSTANCE_SIGNATURE is unavailable' \
          >&2
        exit 1
      fi

      event_socket="$runtime_dir/hypr/$HYPRLAND_INSTANCE_SIGNATURE/.socket2.sock"

      while [ ! -S "$event_socket" ]; do
        sleep 1
      done

      update_focus

      while true; do
        while IFS= read -r event; do
          case "$event" in
            activewindowv2'>>'*)
              update_focus
              ;;
          esac
        done < <(
          socat -U - "UNIX-CONNECT:$event_socket" 2>/dev/null
        )

        sleep 1
      done
    '';
  };

in
{
  # Generic GjallarOS application launch boundary.
  _module.args.gjallarRun = gjallarRun;

  home.packages = [
    gjallarRun
  ];

  # Runtime QoS changes controller values only. Applications stay in the
  # systemd scope in which they were originally launched.
  systemd.user.services.gjallar-resource-qos = {
    Unit = {
      Description = "GjallarOS interactive application QoS";
      After = [ "graphical-session.target" ];
      PartOf = [ "graphical-session.target" ];
    };

    Service = {
      Type = "simple";
      ExecStart = "${gjallarResourceQos}/bin/gjallar-resource-qos";
      Restart = "on-failure";
      RestartSec = 2;
    };

    Install.WantedBy = [ "graphical-session.target" ];
  };
}
