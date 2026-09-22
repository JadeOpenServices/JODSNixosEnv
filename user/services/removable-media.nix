{
  lib,
  pkgs,
  settings,
  ...
}:
let
  usbguard = settings.usbguardEnable or false;
in
lib.mkIf usbguard {
  home.packages = [
    pkgs.zenity
  ];

  services.udiskie = {
    enable = true;
    automount = true;
    notify = true;
    tray = "auto";
  };

}
