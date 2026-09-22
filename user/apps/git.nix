{
  config,
  pkgs,
  lib,
  settings,
  ...
}:

{
  programs.git = {
    enable = true;
    settings = {
      url = {
        "git@github.com:tarantool" = {
          insteadOf = "https://github.com/tarantool";
        };
      };
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
