{ config, lib, pkgs, modulesPath, releaseVersion, ... }:
let
  gjallarctl = pkgs.callPackage ../../pkgs/gjallarctl { };
in
{
  imports = [ "${modulesPath}/installer/cd-dvd/installation-cd-minimal.nix" ];

  image.fileName = lib.mkForce "gjallar-recovery-${config.system.nixos.label}-${pkgs.stdenv.hostPlatform.system}.iso";
  isoImage.squashfsCompression = "zstd -Xcompression-level 15";
  boot.zfs.forceImportRoot = false;

  networking = {
    hostName = "gjallar-recovery";
    networkmanager.enable = true;
    firewall.enable = true;
  };
  services.openssh.enable = lib.mkForce false;
  programs.ssh.startAgent = false;

  environment.systemPackages = with pkgs; [
    age
    cryptsetup
    curl
    efibootmgr
    gitMinimal
    jq
    nixos-install-tools
    openssl
    sbctl
    systemd
    tpm2-tools
    util-linux
    gjallarctl
  ];

  nix.settings = {
    experimental-features = [ "nix-command" "flakes" ];
    require-sigs = true;
  };

  environment.etc."gjallar/recovery-image".text = ''
    GjallarOS minimal recovery and installer environment.
    Verify the signed release manifest before copying this image to recovery media.
    Network access is opt-in; SSH is disabled.
  '';

  # Sourced from deployment/release-policy.json by the root flake. The
  # recovery image can therefore never silently drift from the main release.
  system.stateVersion = releaseVersion;
}
