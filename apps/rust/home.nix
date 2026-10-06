{ config, lib, pkgs, ... }:
lib.mkIf config.gjallar.apps.rust.enable {
  home.packages = with pkgs; [
    cargo
    rustc
    rustfmt
    clippy
    rust-analyzer
    pkg-config
    gcc
  ];
}
