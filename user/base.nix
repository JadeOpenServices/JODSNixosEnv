{
  config,
  settings,
  ...
}:
{
  imports = [
    ../themes/lib/common.nix
    ../themes/lib/home.nix
    ./virtualization
    ./shells/${settings.shell}.nix
    ./apps
  ]
  ++ (map (wm: ./wm/${wm}) settings.wms)
  ++ (map (editor: ./editors/${editor}) settings.editors)
  ++ (map (browser: ./browsers/${browser}.nix) settings.browsers);

  home = {
    username = settings.username;
    homeDirectory = "/home/${settings.username}";
  };

  xdg = {
    enable = true;

    userDirs = {
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
      // (if settings.dotfilesDir != "" then { DOTFILES = settings.dotfilesDir; } else { });
    };
  };

  home.sessionVariables = {
    EDITOR = settings.preferredEditor;
    BROWSER = settings.preferredBrowser;
  };

  programs.home-manager.enable = true;
}
