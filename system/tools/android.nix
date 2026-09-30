{ pkgs, ... }:
{
  # adb and fastboot. systemd's uaccess rules grant the logged-in user access
  # to Android devices, so no adbusers group or extra udev rules are needed.
  # USBGuard still decides whether the phone may connect at all.
  environment.systemPackages = [ pkgs.android-tools ];
}
