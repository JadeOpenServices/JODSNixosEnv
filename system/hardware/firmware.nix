{ ... }:
{
  hardware.enableRedistributableFirmware = true;
  hardware.wirelessRegulatoryDatabase = true;

  services.fwupd.enable = true;

  systemd.services.fwupd-refresh.enable = false;
  systemd.timers.fwupd-refresh.enable = false;
}
