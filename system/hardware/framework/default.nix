{ lib, pkgs, settings, ... }:
if settings.frameworkEnable then
let
    profile = builtins.fromJSON (builtins.readFile (./profiles + "/${settings.frameworkModel}.json"));
    configJson = builtins.toJSON { defaultStrategy = profile.defaultStrategy; strategyOnDischarging = profile.strategyOnDischarging; strategies = profile.strategies; };
in {
    environment.etc."fw-fanctrl/config.json".text = configJson;
    environment.systemPackages = [ pkgs.fw-fanctrl ];
    systemd.services.gjallar-framework-fan-curve = {
        description = "Framework fan curve control";
        wantedBy = [ "multi-user.target" ];
        serviceConfig = { Type = "simple"; Restart = "always"; RestartSec = "2s"; ExecStart = "${pkgs.fw-fanctrl}/bin/fw-fanctrl run --config /etc/fw-fanctrl/config.json --silent ${profile.defaultStrategy}"; };
    };
    systemd.services.gjallar-framework-battery-threshold = {
        description = "Apply Framework battery charge thresholds";
        wantedBy = [ "multi-user.target" ];
        serviceConfig.Type = "oneshot";
        script = ''
            set -eu
            for battery in /sys/class/power_supply/BAT*; do
                [ -d "$battery" ] || continue
                [ ! -w "$battery/charge_control_start_threshold" ] || printf '%s\n' ${toString profile.batteryStart} > "$battery/charge_control_start_threshold"
                [ ! -w "$battery/charge_control_end_threshold" ] || printf '%s\n' ${toString profile.batteryEnd} > "$battery/charge_control_end_threshold"
            done
        '';
    };
}
else { }
