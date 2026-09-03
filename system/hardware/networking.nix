{ lib, settings, ... }:
{
  networking.networkmanager.enable = true;
  # Disable Wi-Fi power saving; it can cause severe throughput/latency
  # regressions on laptop chipsets while still allowing suspend normally.
  networking.networkmanager.wifi.powersave = false;
  hardware.enableRedistributableFirmware = true;
  hardware.wirelessRegulatoryDatabase = true;
}
