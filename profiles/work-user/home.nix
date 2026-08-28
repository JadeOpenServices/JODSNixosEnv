{ lib, settings, ... }:
{
  imports = [ ../work/home.nix ];
  home.username = lib.mkForce settings.workUsername;
  home.homeDirectory = lib.mkForce "/home/${settings.workUsername}";
}
