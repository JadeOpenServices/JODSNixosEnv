{
  pkgs,
  ...
}:
{
  imports = [
    ../../user/base.nix
    ../../user/apps/spotify.nix
    ../../user/apps/btop
    ../../user/apps/khal.nix
    ../../user/apps/neofetch
    ../../user/gaming/nethack.nix
    ../../user/apps/tlaplus.nix
    ../../user/apps/latex.nix
  ];

  nix.package = pkgs.nix;

  home.packages = with pkgs; [
    libreoffice-fresh
    yubikey-manager
    obs-studio
    tty-clock
    teleport
    rtorrent
    tigervnc
    swayimg
    openvpn
    update-resolv-conf
    drawio
    gimp
    mpv
    chromium
  ];

  home.stateVersion = "23.05";
}
