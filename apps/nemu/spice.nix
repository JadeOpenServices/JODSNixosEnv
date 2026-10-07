{
  config,
  lib,
  pkgs,
  ...
}:
lib.mkIf config.gjallar.apps.nemu.enable {
  virtualisation.spiceUSBRedirection.enable = true;
  environment.systemPackages = with pkgs; [
    spice-gtk
  ];
}
