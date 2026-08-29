{ lib, settings, ... }:
{
    # Laptop related settings for optimization.
    powerManagement.enable = true;
    services.thermald.enable = true;
    services.auto-cpufreq.enable = true;
    services.auto-cpufreq.settings = {
        battery = {
            governor = "powersave";
            turbo = "never";
        };
        charger = {
            governor = "performance";
            turbo = "auto";
        };
    };

    systemd.services.gjallar-battery-charge-threshold = {
        description = "Apply GjallarOS battery charge thresholds";
        wantedBy = [ "multi-user.target" ];
        after = [ "local-fs.target" ];
        serviceConfig.Type = "oneshot";
        script = ''
            set -eu
            for battery in /sys/class/power_supply/BAT*; do
                [ -d "$battery" ] || continue
                if [ -w "$battery/charge_control_start_threshold" ]; then
                    printf '%s\n' 75 > "$battery/charge_control_start_threshold"
                fi
                if [ -w "$battery/charge_control_end_threshold" ]; then
                    printf '%s\n' 95 > "$battery/charge_control_end_threshold"
                fi
            done
        '';
    };

    services.logind.settings.Login = {
        # A short press is safe; require a long press for poweroff.
        HandlePowerKey = "suspend";
        HandlePowerKeyLongPress = "poweroff";
    } // lib.optionalAttrs (settings.clamshellEnable or true) {
        # Suspend-to-RAM when undocked; stay awake with an external display.
        HandleLidSwitch = "suspend";
        HandleLidSwitchExternalPower = "ignore";
        HandleLidSwitchDocked = "ignore";
    };


    services.upower.enable = true;
}
