{
  inputs,
  pkgs,
  ...
}:
{
  imports = [
    ../../user/base.nix
    ../../user/apps/mangohud.nix
    ../../user/apps/kdeconnect.nix
    ../../user/apps/ssh.nix
    ../../user/gaming/nethack.nix
    ../../user/gaming/oss-games.nix
    ../../user/gaming/steam.nix
    ../../user/gaming/lutris.nix
    ../../user/apps/tlaplus.nix
    ../../user/apps/latex.nix
    ../../user/apps/btop
    ../../user/apps/mpd
  ];

  nix.package = pkgs.nix;

  home.packages = with pkgs; [
    obs-studio
    lshw
    calc
    sysstat
    gnupg
    libreoffice-fresh
    qbittorrent
    cpulimit
    swayimg
    vesktop
    telegram-desktop
    drawio
    gimp
    mpv
    ghostty
    kicad
    pcsx2
    dolphin-emu
    pwgen
    element-desktop
    gnome-keyring
    seahorse
    dmidecode
    sysbench
    openvpn
    update-resolv-conf
    chromium
    unzip
    translate-shell
    android-tools
    jq

    inputs.late.packages.${pkgs.stdenv.hostPlatform.system}.late
  ];

  home.stateVersion = "24.11";
}
