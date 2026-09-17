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
  # User-space USB authorization UI is deliberately request-only.
  # The future root-owned gjallar-usbtrust service is the sole
  # authorization and persistent-trust mutation owner.
  home.packages = [
    pkgs.zenity
  ];
}
