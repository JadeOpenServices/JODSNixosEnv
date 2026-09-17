{
  config,
  lib,
  pkgs,
  settings,
  ...
}:
let
  batteryAvailable = lib.attrByPath [
    "class"
    "capabilities"
    "battery"
  ] false config.oddc.resolved;

  chargeThresholdsSupported = lib.attrByPath [
    "capabilities"
    "chargeThresholds"
  ] false config.oddc.resolved;

  batteryCountdown = pkgs.writeShellApplication {
    name = "gjallar-battery-countdown";
    runtimeInputs = [ pkgs.zenity ];
    text = ''
      set -euo pipefail

      produce_status() {
        while :; do
          capacity=100
          status=Unknown
          found=0

          for battery in /sys/class/power_supply/BAT*; do
            [ -r "$battery/capacity" ] || continue
            found=1
            value="$(cat "$battery/capacity")"
            [ "$value" -lt "$capacity" ] && capacity="$value"
            [ -r "$battery/status" ] && status="$(cat "$battery/status")"
          done

          [ "$found" -eq 1 ] || return 0
          [ "$status" = Discharging ] || return 0
          [ "$capacity" -gt 2 ] || return 0
          [ "$capacity" -le 5 ] || return 0

          progress=$(( (capacity - 2) * 100 / 3 ))
          printf '%s\n' "$progress"
          printf '# Battery: %s%%\nEmergency shutdown at 2%%. Connect power now.\n' "$capacity"
          sleep 10
        done
      }

      produce_status | zenity --progress \
        --title="GjallarOS Battery" \
        --width=520 \
        --no-cancel \
        --auto-close \
        --percentage=100
    '';
  };

  batteryGuard = pkgs.writeShellApplication {
    name = "gjallar-battery-guard";
    runtimeInputs = with pkgs; [
      coreutils
      plymouth
      systemd
      util-linux
      zenity
    ];
    text = ''
      set -euo pipefail

      warned10=0
      warned5=0

      launch_warning() {
        level="$1"
        message="$2"
        runtime="/run/user/$(id -u ${lib.escapeShellArg settings.username})"
        [ -S "$runtime/bus" ] || return 0

        unit="gjallar-battery-$level-$(date +%s%N)"
        runuser -u ${lib.escapeShellArg settings.username} -- env \
          XDG_RUNTIME_DIR="$runtime" \
          DBUS_SESSION_BUS_ADDRESS="unix:path=$runtime/bus" \
          systemd-run --user --quiet --collect --unit="$unit" \
            zenity --warning \
              --title="GjallarOS Battery" \
              --width=520 \
              --text="$message" \
              >/dev/null 2>&1 || true
      }

      launch_countdown() {
        runtime="/run/user/$(id -u ${lib.escapeShellArg settings.username})"
        [ -S "$runtime/bus" ] || return 0

        runuser -u ${lib.escapeShellArg settings.username} -- env \
          XDG_RUNTIME_DIR="$runtime" \
          DBUS_SESSION_BUS_ADDRESS="unix:path=$runtime/bus" \
          systemd-run --user --quiet --collect \
            --unit=gjallar-battery-countdown \
            ${batteryCountdown}/bin/gjallar-battery-countdown \
            >/dev/null 2>&1 || true
      }

      while :; do
        capacity=100
        status=Unknown
        found=0

        for battery in /sys/class/power_supply/BAT*; do
          [ -r "$battery/capacity" ] || continue
          found=1
          value="$(cat "$battery/capacity")"
          [ "$value" -lt "$capacity" ] && capacity="$value"
          [ -r "$battery/status" ] && status="$(cat "$battery/status")"
        done

        if [ "$found" -eq 0 ] || [ "$status" != Discharging ]; then
          warned10=0
          warned5=0
          sleep 20
          continue
        fi

        if [ "$capacity" -le 2 ]; then
          plymouth --show-splash >/dev/null 2>&1 || true
          plymouth change-mode --shutdown >/dev/null 2>&1 || true
          systemctl poweroff
          exit 0
        fi

        if [ "$capacity" -le 5 ] && [ "$warned5" -eq 0 ]; then
          warned5=1
          launch_warning critical \
            "<b>Critical battery: $capacity%</b>\n\nGjallarOS will shut down automatically at 2% to protect the battery. Connect power now."
          launch_countdown
        elif [ "$capacity" -le 10 ] && [ "$warned10" -eq 0 ]; then
          warned10=1
          launch_warning low \
            "<b>Low battery: $capacity%</b>\n\nConnect power soon. A shutdown countdown starts at 5%, with emergency shutdown at 2%."
        fi

        sleep 20
      done
    '';
  };
in
{
  config = lib.mkIf batteryAvailable (lib.mkMerge [
    {
      powerManagement.enable = true;
      services.upower.enable = true;

      services.logind.settings.Login = {
        HandlePowerKey = "suspend";
        HandlePowerKeyLongPress = "poweroff";
      }
      // lib.optionalAttrs settings.clamshellEnable {
        HandleLidSwitch = "suspend";
        HandleLidSwitchExternalPower = "ignore";
        HandleLidSwitchDocked = "ignore";
      };

      systemd.services.gjallar-battery-guard = {
        description = "GjallarOS low-battery warnings and emergency shutdown";
        wantedBy = [ "multi-user.target" ];
        after = [
          "systemd-logind.service"
          "upower.service"
        ];
        serviceConfig = {
          ExecStart = lib.getExe batteryGuard;
          Restart = "always";
          RestartSec = 5;
          StandardOutput = "journal";
          StandardError = "journal";
        };
      };
    }

    (lib.mkIf chargeThresholdsSupported {
      systemd.services.gjallar-battery-charge-threshold = {
        description = "Apply GjallarOS battery charge thresholds";
        wantedBy = [ "multi-user.target" ];
        after = [ "local-fs.target" ];
        serviceConfig.Type = "oneshot";
        script = ''
          start_applied=0
          end_applied=0

          for battery in /sys/class/power_supply/BAT*; do
            [ -d "$battery" ] || continue

            if [ -w "$battery/charge_control_start_threshold" ]; then
              printf '%s\n' 75 > "$battery/charge_control_start_threshold"
              start_applied=1
            elif [ -w "$battery/charge_start_threshold" ]; then
              printf '%s\n' 75 > "$battery/charge_start_threshold"
              start_applied=1
            fi

            if [ -w "$battery/charge_control_end_threshold" ]; then
              printf '%s\n' 95 > "$battery/charge_control_end_threshold"
              end_applied=1
            elif [ -w "$battery/charge_stop_threshold" ]; then
              printf '%s\n' 95 > "$battery/charge_stop_threshold"
              end_applied=1
            fi
          done

          if [ "$start_applied" -ne 1 ] || [ "$end_applied" -ne 1 ]; then
            echo "ODDC declares charge-threshold support, but writable start/end controls are unavailable" >&2
            exit 1
          fi
        '';
      };
    })
  ]);
}
