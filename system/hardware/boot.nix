{
  settings,
  ...
}:
{
  # GjallarOS installations are UEFI-only. Secure Boot replaces systemd-boot
  # with Lanzaboote through system/security/secure-boot.
  boot.loader.efi.canTouchEfiVariables = true;
  boot.loader.systemd-boot.enable = true;

  # Normal boots stay hidden; diagnostics expose the generation chooser.
  boot.loader.timeout = if settings.debugFunctions then 5 else 0;
}
