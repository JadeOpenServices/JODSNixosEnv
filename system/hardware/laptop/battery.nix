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

    systemd.services.alfheim-battery-charge-threshold = {
        description = "Apply AlfheimOS battery charge thresholds";
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

    services.logind = {
        # A short press is safe; require a long press for poweroff.
        powerKey = "suspend";
        powerKeyLongPress = "poweroff";
        # Suspend-to-RAM when undocked; stay awake with an external display.
        lidSwitch = "suspend";
        lidSwitchExternalPower = "ignore";
        lidSwitchDocked = "ignore";
    };

    services.upower.enable = true;
}
