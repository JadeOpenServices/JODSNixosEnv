{ config, lib, pkgs, ... }:
lib.mkIf config.gjallar.apps.libreoffice.enable {
  home.packages = [ pkgs.libreoffice-fresh ];
}
