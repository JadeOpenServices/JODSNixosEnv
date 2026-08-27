{ lib, settings, ... }:
{
    networking.networkmanager.enable = true;
    hardware.enableRedistributableFirmware = true;
    hardware.wirelessRegulatoryDatabase = true;
    boot.kernelModules = lib.optional (settings.wifiDriver != "") settings.wifiDriver;
}
