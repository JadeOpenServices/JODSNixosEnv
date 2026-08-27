{pkgs, inputs, ...}:
rec {
    # Basic configuration.
    system = "x86_64-linux";
    profile = "desktop"; # Select from profiles directory
    hostname = "gjallar"; # Hostname
    username = "user"; # Username (set by the installer)
    timezone = "Europe/Moscow"; # Select timezone
    locale = "en_US.UTF-8"; # Select locale
    name = "Your Name"; # Name (git config)
    email = "user@example.com"; # Email (git config)
    dotfilesDir = "/home/${username}/.dotfiles"; # Absolute path of the repo;
    workUserEnable = false;
    workUsername = "";
    dockerEnable = false;

    # App configurations.
    shell = "zsh"; # See user/shells directory.
    editors = ["neovim"]; # See user/editors directory.
    browsers = ["librewolf" "qutebrowser" "zen-browser"]; # See user/browsers directory.
    preferredEditor = "nvim"; # Session variable $TERM.
    preferredBrowser = "librewolf"; # Session variable $BROWSER.

    # Optional music scrobbling (configured by the installer).
    enableScrobbling = false;
    enableLastfm = false;
    enableListenbrainz = false;
    lastfmUsername = "";
    listenbrainzUsername = "";
    frameworkEnable = false;
    frameworkModel = "";
    graphicsVendor = "amd";
    graphicsType = "integrated";
    graphicsCompute = false;
    graphicsBusId = "";
    graphicsIntegratedBusId = "";
    wifiDriver = "";
    aiModel = "qwen2.5-coder:7b";
    aiContextTokens = 8192;
    aiVramMB = 0;
    nemuEnable = false;
    nemuGpuPassthrough = false;
    nemuGpuIds = [];
    luksTpm2Enable = false;

    # WM and theming.
    wms = ["hyprland" "sway"]; # See user/wm/ and system/wm directories.
    theme = "morpho"; # Selected theme from themes directory (./themes/)

    # Do not change these!
    profileDetails = import (./. + "/profiles/${profile}/details.nix") {};
    themeDetails = import (./. + "/themes/${theme}.nix") {inherit pkgs;};
}
