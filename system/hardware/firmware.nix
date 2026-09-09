{ ... }:
{
  # Firmware updates are available through fwupdmgr and the graphical tools.
  services.fwupd.enable = true;

  # fwupd 2.1.4 / NixOS 26.05: the automatic refresh unit can
  # fail during activation when PolicyKit is temporarily unavailable.
  # Keep fwupd itself enabled; disable only background refresh.
  systemd.services.fwupd-refresh.enable = false;
  systemd.timers.fwupd-refresh.enable = false;
}
