{ pkgs, ... }:
{
    programs.vscodium = {
        enable = true;
        profiles.default.extensions = with pkgs.vscode-extensions; [
            golang.go
            ms-python.python
            redhat.vscode-yaml
            tamasfe.even-better-toml
            pkief.material-icon-theme
            eamodio.gitlens
            usernamehw.errorlens
            streetsidesoftware.code-spell-checker
            editorconfig.editorconfig
            hashicorp.terraform
        ];
        profiles.default.userSettings = {
            # Keep standard editor shortcuts such as Ctrl+C/Ctrl+V. The Vim
            # extension changes modal input and can look like overwrite mode.
            "editor.overwrite" = false;
            # Keep search, file watching, and AI-assisted indexing focused on
            # source files instead of generated/dependency trees.
            "search.followSymlinks" = false;
            "search.useIgnoreFiles" = true;
            "files.watcherExclude" = {
                "**/.git/**" = true;
                "**/node_modules/**" = true;
                "**/dist/**" = true;
                "**/build/**" = true;
                "**/target/**" = true;
                "**/out/**" = true;
                "**/.direnv/**" = true;
                "**/.next/**" = true;
                "**/.turbo/**" = true;
                "**/.venv/**" = true;
                "**/result/**" = true;
            };
            "search.exclude" = {
                "**/.git" = true;
                "**/node_modules" = true;
                "**/dist" = true;
                "**/build" = true;
                "**/target" = true;
                "**/out" = true;
                "**/.direnv" = true;
                "**/.next" = true;
                "**/.turbo" = true;
                "**/.venv" = true;
                "**/result" = true;
                "**/.env*" = true;
                "**/*secret*" = true;
                "**/*.key" = true;
            };
            "files.exclude" = {
                "**/.direnv" = true;
                "**/node_modules" = true;
                "**/result" = true;
                "**/.env*" = true;
                "**/*secret*" = true;
                "**/*.key" = true;
                "**/target" = true;
            };
            "git.confirmSync" = false;
            "git.autofetch" = true;
            "git.enableSmartCommit" = true;
            "npm.autoDetect" = "off";
        };
    };
}
