{ lib, settings, ... }:
{
    networking.networkmanager.enable = true;
    hardware.enableRedistributableFirmware = true;
    hardware.wirelessRegulatoryDatabase.enable = true;
    boot.kernelModules = lib.optional (settings.wifiDriver != "") settings.wifiDriver;
}
