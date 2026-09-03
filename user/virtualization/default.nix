{ lib, settings, ... }:
{
  imports = [
    ./winapps
    ./quickemu.nix
  ]
  ++ lib.optional settings.nemuEnable ./nemu.nix;
}
