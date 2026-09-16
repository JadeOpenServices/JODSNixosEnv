{
  lib,
  pkgs,
  settings,
  ...
}:
{
  imports = [
    ../../system/base.nix
    ./hardware-configuration.nix

    ../../system/hardware/desktop/mouse.nix
    ../../system/hardware/desktop/boot.nix
    ../../system/hardware/desktop/nfs.nix
    ../../system/security/desktop/firewall.nix
    ../../system/security/vpn/xray.nix
    ../../system/security/ssh.nix
    ../../system/security/sops.nix
    ../../system/apps/thunar.nix
    ../../system/apps/guix.nix
    ../../system/apps/platformio.nix
    ../../system/gaming/steam.nix
  ];

  networking.networkmanager.dns = "dnsmasq";

  nix.settings.trusted-users = [ "root" ];

  users.users.${settings.username}.extraGroups = [
    "networkmanager"
    "gamemode"
    "dialout"
  ]
  ++ lib.optionals (!settings.endpointManagedDevice) [ "wheel" ];

  services.emacs.enable = true;

  environment.systemPackages = with pkgs; [
    usbutils
    inetutils
  ];

  fonts.packages = [ settings.themeDetails.fontPkg ];

  system.stateVersion = "24.11";
}
