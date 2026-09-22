{
  inputs,
  config,
  pkgs,
  ...
}:

{
  home.packages = with pkgs; [
    gnome-tweaks
    gnomeExtensions.appindicator
    gnomeExtensions.blur-my-shell
    gnomeExtensions.useless-gaps
  ];

  dconf.settings = {
    "org/gnome/shell" = {
      disable-user-extensions = false;
      enabled-extensions = [
        "blur-my-shell@aunetx"
        "appindicatorsupport@rgcjonas.gmail.com"
        "flypie@schneegans.github.com" # Must be downloaded manually!
        "user-theme@gnome-shell-extensions.gcampax.github.com"
        "useless-gaps@pimsnel.com" # Waiting to be updated at Nix. Manually.
      ];
    };

    "org/gnome/shell/extensions/user-theme" = {
    };
    "/org/gnome/shell/extensions/flypie" = {
    };
    "/org/gnome/shell/extensions/blur-my-shell/overview" = {
      style-components = 3;
    };
  };
}
