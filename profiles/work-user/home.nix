{ settings, ... }:
{
  imports = [ ../work/home.nix ];
  home.username = settings.workUsername;
  home.homeDirectory = "/home/${settings.workUsername}";
}
