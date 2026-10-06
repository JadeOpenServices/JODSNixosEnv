{
  config,
  lib,
  pkgs,
  ...
}:
lib.mkIf config.gjallar.apps.qbittorrent.enable {
  home.packages = [ pkgs.qbittorrent ];
}
