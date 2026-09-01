{ pkgs, ... }:

{
  packages = [
    pkgs.python3Packages.black
  ];
}
