{ lib, settings, ... }:
let
  workSettings = settings // {
    username = settings.workUsername;
    dotfilesDir = "/home/${settings.workUsername}/.dotfiles";
  };
in
{
  # The work account reuses the work profile, but all path-derived settings
  # must resolve inside /home/<work user>, never the primary user's home.
  _module.args.settings = lib.mkForce workSettings;
  imports = [ ../work/home.nix ];
  home.username = lib.mkForce settings.workUsername;
  home.homeDirectory = lib.mkForce "/home/${settings.workUsername}";
}
