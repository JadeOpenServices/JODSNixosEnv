{ config, lib, pkgs, ... }:
lib.mkIf config.gjallar.apps.gimp.enable {
  home.packages = [ pkgs.gimp ];
}
