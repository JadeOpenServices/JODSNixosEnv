{
  config,
  pkgs,
  lib,
  settings,
  ...
}:

lib.mkIf config.gjallar.apps.git.enable {
  programs.git = {
    enable = true;
    settings = {
      core.editor = lib.getExe pkgs.${settings.preferredEditor};
      user = {
        name = settings.name;
        email = settings.email;
      };
      stash = {
        showPatch = true;
      };
    };
  };

  programs.lazygit = {
    enable = true;
    settings = {
      gui = {
        switchTabsWithPanelJumpKeys = true;
      };
    };
  };
}
