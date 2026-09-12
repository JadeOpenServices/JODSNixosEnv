{
  config,
  pkgs,
  settings,
  ...
}:
{
  imports = [
    ../../themes/lib/common.nix
    ../../themes/lib/home.nix
    ../../user/apps/spotify.nix
    ../../user/apps/btop
    ../../user/apps/khal.nix
    ../../user/apps/neofetch
    ../../user/gaming/nethack.nix
    ../../user/apps/tlaplus.nix
    ../../user/apps/latex.nix
    ../../user/shells/${settings.shell}.nix
    ../../user/virtualization
    ../../user/apps
  ]
  ++ (map (wm: ../../user/wm/${wm}) settings.wms)
  ++ (map (editor: ../../user/editors/${editor}) settings.editors)
  ++ (map (browser: ../../user/browsers/${browser}.nix) settings.browsers);

  nix.package = pkgs.nix;
  home = {
    username = settings.username;
    homeDirectory = "/home/${settings.username}";
  };

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

    # Sometimes needed for work.
    chromium
  ];

  xdg.enable = true;
  xdg.userDirs = {
    enable = true;
    createDirectories = true;
    music = "${config.home.homeDirectory}/Media/Music";
    videos = "${config.home.homeDirectory}/Media/Videos";
    pictures = "${config.home.homeDirectory}/Media/Pictures";
    download = "${config.home.homeDirectory}/Downloads";
    documents = "${config.home.homeDirectory}/Documents";
    templates = null;
    desktop = null;
    publicShare = null;
    extraConfig = {
      BOOK = "${config.home.homeDirectory}/Media/Books";
    }
    // (
      if settings.dotfilesDir != "" then
        {
          DOTFILES = settings.dotfilesDir;
        }
      else
        { }
    );
  };

  home.sessionVariables = {
    EDITOR = settings.preferredEditor;
    BROWSER = settings.preferredBrowser;
  };

  programs.home-manager.enable = true;
  home.stateVersion = "23.05";
}
