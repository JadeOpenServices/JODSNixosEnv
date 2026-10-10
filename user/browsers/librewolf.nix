{
  config,
  lib,
  pkgs,
  ...
}:
let
  profileName = "Default";

  # Copies the LibreWolf profile into the Firefox profile. LibreWolf is only
  # read; the Firefox profile is saved to a tarball first.
  import' = pkgs.writers.writePython3Bin "gjallar-librewolf-import" {
    flakeIgnore = [ "E501" ];
  } (builtins.readFile ./librewolf-import.py);
in
{
  home.packages = lib.mkIf config.programs.firefox.enable [
    import'
    (pkgs.writeTextDir "share/zsh/site-functions/_gjallar-librewolf-import" ''
      #compdef gjallar-librewolf-import
      _arguments \
        '--dry-run[list the files without copying]' \
        '--src[LibreWolf profile]:directory:_files -/' \
        '--dst[Firefox profile]:directory:_files -/'
    '')
  ];

  stylix.targets.librewolf.profileNames = [ profileName ];
  stylix.targets.librewolf.colorTheme.enable = true;
  programs.librewolf = {
    enable = true;
    settings = {
      "privacy.clearOnShutdown.history" = false;
      "privacy.clearOnShutdown.sessions" = false;
      "privacy.clearOnShutdown.downloads" = false;
      "privacy.clearOnShutdown.cookies" = false;
      "privacy.clearOnShutdown.formdata" = false;
      "privacy.clearOnShutdown.cache" = false;

      "privacy.clearOnShutdown_v2.cookiesAndStorage" = false;
      "privacy.clearOnShutdown_v2.browsingHistoryAndDownloads" = false;
      "privacy.clearOnShutdown_v2.formdata" = false;
      "privacy.clearOnShutdown_v2.cache" = false;
      "privacy.clearOnShutdown_v2.historyFormDataAndDownloads" = false;
      "privacy.clearOnShutdown_v2.siteSettings" = false;
    };
  };

  programs.librewolf.profiles.${profileName} = {
    extensions = {
      force = true;
      packages = with pkgs.nur.repos.rycee.firefox-addons; [
        nextcloud-passwords
        ublock-origin
        vimium
        stylus
      ];
    };
  };
}
