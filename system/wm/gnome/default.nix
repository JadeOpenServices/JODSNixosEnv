{ pkgs, ... }:
{
  imports = [ ../common/wayland.nix ];
  services.udev.packages = [ pkgs.gnome-settings-daemon ];
}
