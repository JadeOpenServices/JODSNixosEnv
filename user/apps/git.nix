{ config, pkgs, settings, ...}:

{
    programs.git = {
        enable = true;
        settings = {
            url = {
                "git@github.com:tarantool" = {
                    insteadOf = "https://github.com/tarantool";
                };
            };
            core.editor = "nvim";
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
