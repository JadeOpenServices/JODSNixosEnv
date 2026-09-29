{
  settings,
  ...
}:
{
  boot.loader.efi.canTouchEfiVariables = true;
  boot.loader.systemd-boot.enable = true;
  # The boot menu editor lets anyone at the keyboard append init=/bin/sh and
  # get a root shell, which also bypasses TPM-unlocked disk encryption.
  boot.loader.systemd-boot.editor = false;

  boot.loader.timeout = if settings.debugFunctions then 5 else 0;
}
