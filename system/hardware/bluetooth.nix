{ config, pkgs, ...}:

{
    environment.systemPackages = with pkgs; [
        bluez
        bluez-tools
    ];

    hardware.bluetooth = {
        settings = {
            General = {
                ControllerMode = "dual";
            };
        };
        enable = true;
        package = pkgs.bluez;
    };
}
