{
  pkgs,
  settings,
  ...
}:
{
  imports = [
    ./hardware/sound.nix
    ./hardware/bluetooth.nix
    ./hardware/graphics
    ./hardware/networking.nix
    ./hardware/firmware.nix
    ./hardware/fingerprint.nix
    ./hardware/input.nix
    ./recovery
    ./virtualization
    ./gaming/nethack.nix
    ../themes/lib/common.nix
    ./tools
    ./users/work.nix
    ./users/privilege.nix
    ./apps/ollama.nix
  ]
  ++ (map (wm: ./wm/${wm}) settings.wms);

  boot.kernelPackages = pkgs.linuxPackages_latest;

  nixpkgs.overlays = import ../pkgs/lib/overlays.nix;
  nixpkgs.config.allowUnfree = true;

  nix.settings.experimental-features = [
    "nix-command"
    "flakes"
  ];

  networking.hostName = settings.hostname;
  networking.networkmanager.enable = true;

  time.timeZone = settings.timezone;
  services.chrony.enable = true;

  i18n.defaultLocale = settings.locale;
  i18n.extraLocaleSettings.LC_ALL = settings.locale;

  programs.${settings.shell}.enable = true;

  users.users.${settings.username} = {
    isNormalUser = true;
    shell = pkgs.${settings.shell};
    description = settings.username;
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
    wget
    git
    vim
  ];

  services.gvfs.enable = true;
}
