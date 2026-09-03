{
  config,
  lib,
  pkgs,
  settings,
  ...
}:

let
  nixpkgs = import <nixpkgs> { config = config.nixpkgs.config; };
in
{

  environment.systemPackages = with pkgs; [
    virt-viewer
    tigervnc
    qemu
    samba
  ];

  imports = [
    ./module.nix
  ];

  programs.nemu = {
    package = pkgs._nemu;
    enable = true;
    vhostNetGroup = "vhost";
    macvtapGroup = "vhost";
    usbGroup = "usb";
    users = {
      ${settings.username} = {
        # Install Nemu without starting its daemon or creating veth
        # devices during boot. Users can start it on demand once a
        # VM definition exists; this keeps normal boots clean.
        autoAddVeth = false;
        autoStartDaemon = false;
        # autoStartVMs = [ "Win11" ];
      };
    }
    // lib.optionalAttrs settings.workUserEnable {
      ${settings.workUsername} = {
        autoAddVeth = false;
        autoStartDaemon = false;
      };
    };
  };
}
