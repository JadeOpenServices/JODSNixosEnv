{ pkgs, ... }:
{
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
