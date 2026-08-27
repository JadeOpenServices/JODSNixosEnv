{ lib, pkgs, settings, ... }:
lib.mkIf settings.workUserEnable {
  users.users.${settings.workUsername} = {
    isNormalUser = true;
    description = "${settings.username} work account";
    shell = pkgs.${settings.shell};
    extraGroups = [ "audio" "video" "networkmanager" ];
  };
}
