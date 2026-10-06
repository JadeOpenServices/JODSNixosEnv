{
  inputs,
  lib,
  pkgs,
  settings,
  installState,
  ...
}:
{
  imports = [
    ../generated/hardware.nix

    ./hardware/boot.nix
    ./hardware/battery.nix
    ./hardware/sound.nix
    ./hardware/bluetooth.nix
    ./hardware/graphics
    ./hardware/networking.nix
    ./hardware/firmware.nix
    ./hardware/fingerprint.nix
    ./hardware/input.nix
    ./hardware/fan-control.nix

    ./security/firewall.nix
    ./security/local-hardening.nix
    ./security/keyring.nix
    ./maintenance/storage.nix
    ./maintenance/installer-resume.nix
    ./services/printing.nix
    ./services/removable-media.nix
    ./services/tailscale.nix

    inputs.oddc.nixosModules.default

    ./recovery
    ./virtualization
    ../themes/lib/common.nix
    ./tools
    ./users/privilege.nix
    ./apps/ollama.nix
  ]
  ++ (map (wm: ./wm/${wm}) settings.wms);

  # The installer's answer: only this machine's model, never the catalog.
  oddc.catalog = lib.mkIf (builtins.pathExists ../generated/oddc) ../generated/oddc;

  oddc.device =
    let
      model = settings.oddcModel or "";
    in
    if model == "" then null else model;

  boot.kernelPackages = pkgs.linuxPackages_latest;

  nixpkgs.overlays = import ../pkgs/lib/overlays.nix;
  nixpkgs.config.allowUnfree = true;

  nix.settings.experimental-features = [
    "nix-command"
    "flakes"
  ];

  networking.hostName = settings.hostname;
  time.timeZone = settings.timezone;
  services.chrony.enable = true;

  i18n.defaultLocale = settings.locale;
  i18n.extraLocaleSettings.LC_ALL = settings.locale;

  programs.${settings.shell}.enable = true;

  users.users.${settings.username} = {
    isNormalUser = true;
    shell = pkgs.${settings.shell};
    description = settings.username;
    extraGroups = [ "networkmanager" ]
    ++ lib.optionals (!settings.endpointManagedDevice) [ "wheel" ];
  };

  programs.nix-ld = {
    enable = true;
    libraries = with pkgs; [ stdenv.cc.cc ];
  };

  environment.systemPackages = with pkgs; [
    home-manager
    nix-index
    pciutils
    go-mtpfs
    lsof
    git
    vim
  ];

  services.gvfs.enable = true;

  system.stateVersion = installState.nixosStateVersion;
}
