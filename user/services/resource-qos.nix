{ pkgs, ... }:

let
  managedWeight = 80;
  foregroundWeight = 100;
  explicitBackgroundWeight = 25;

  gjallarRun = pkgs.writeShellApplication {
    name = "gjallar-run";

    runtimeInputs = [
      pkgs.app2unit
    ];

    text = ''
      family="desktop"

      if [ "$#" -ge 2 ] && [ "$1" = "--family" ]; then
        family="$2"
        shift 2
      fi

      if [ "$#" -eq 0 ]; then
        echo "usage: gjallar-run [--family desktop|jods|ai|background] COMMAND [ARG...]" >&2
        exit 64
      fi

      command_name="''${1##*/}"

      case "$family" in
        desktop)
          app_name="gjallar-$command_name"
          slice="a"
          weight=${toString managedWeight}
          ;;
        jods)
          app_name="jods-$command_name"
          slice="a"
          weight=${toString managedWeight}
          ;;
        ai)
          app_name="gjallar-ai-$command_name"
          slice="a"
          weight=${toString managedWeight}
          ;;
        background)
          app_name="background-$command_name"
          slice="b"
          weight=${toString explicitBackgroundWeight}
          ;;
        *)
          echo "gjallar-run: unknown workload family: $family" >&2
          exit 64
          ;;
      esac

      exec app2unit \
        -s "$slice" \
        -a "$app_name" \
        -p "CPUWeight=$weight" \
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
      unfocused_weight=${toString managedWeight}
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
        # Only the desktop family is focus-sensitive. JODS, AI and explicit
        # background workloads keep the policy assigned when they start.
        case "$unit" in
          app-*-gjallar\\x2dai\\x2d*.scope|app-*-gjallar-ai-*.scope)
            return 1
            ;;
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
          set_weight "$current_unit" "$unfocused_weight"
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
  # One app2unit launch boundary, with identity families kept separate from
  # policy. Desktop focus policy applies only to gjallar-* scopes.
  #
  #   desktop    -> gjallar-*    -> app.slice, CPUWeight 80/100 by focus
  #   jods       -> jods-*       -> app.slice, CPUWeight 80
  #   ai         -> gjallar-ai-* -> app.slice, CPUWeight 80
  #   background -> background-* -> background.slice, CPUWeight 25
  #
  # Existing JODS and Gjallar AI system services keep their own systemd
  # ownership. These families are for app2unit-launched user workloads.
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
