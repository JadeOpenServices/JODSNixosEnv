{
  config,
  pkgs,
  settings,
  ...
}:

{
  boot.loader.efi.canTouchEfiVariables = true;
  boot.loader.grub = {
    enable = true;
    device = "nodev";
    efiSupport = true;
  };

  boot.loader.timeout = if settings.debugFunctions then 5 else 0;
}
