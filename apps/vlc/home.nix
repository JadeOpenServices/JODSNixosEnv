{
  config,
  lib,
  pkgs,
  ...
}:
lib.mkIf config.gjallar.apps.vlc.enable {
  home.packages = [ pkgs.vlc ];
}
