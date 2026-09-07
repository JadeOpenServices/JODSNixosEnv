{ pkgs, ... }:
{
  # Complete local Rust build/deployment baseline. Cargo and rustc are
  # separate Nix packages; GCC supplies the native linker required by many
  # crates and test binaries.
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
