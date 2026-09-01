{ pkgs, ... }:

{
  packages = [
    pkgs.python3Packages.httpx
  ];
}
