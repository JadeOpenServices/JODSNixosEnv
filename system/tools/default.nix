{ pkgs, settings, ... }:
{
    imports = [ ./scripts/default.nix ];
    programs.nh = {
        enable = true;
        clean.enable = false;
        flake = settings.dotfilesDir;
    };
    environment.systemPackages = with pkgs; [
        nix-output-monitor
        nvd
        lm_sensors
        procps
        gawk
        findutils
        openssl
        power-profiles-daemon
    ];
}
