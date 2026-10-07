# qutebrowser, selectable through `browsers` in user.config.json.
{
  programs.qutebrowser = {
    enable = true;
    loadAutoconfig = true;
    searchEngines = {
      DEFAULT = "https://duckduckgo.com/?q={}";
      nix = "https://search.nixos.org/packages?query={}";
      opt = "https://search.nixos.org/options?query={}";
      gh = "https://github.com/search?q={}";
    };
    settings = {
      auto_save.session = true;
      scrolling.smooth = true;
      zoom.default = "110%";
    };
  };
}
