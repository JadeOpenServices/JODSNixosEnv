{
  config,
  lib,
  pkgs,
  settings,
  ...
}:
let
  ghosttyShell = pkgs.writeShellScript "gjallar-ghostty-shell" ''
    if [ -t 1 ]; then
      printf '\n'
      ${lib.getExe pkgs.fastfetch} || true
      printf '\n'
    fi

    exec ${lib.getExe pkgs.${settings.shell}} "$@"
  '';
in
{
  home.activation.gjallarGhosttyNoctaliaFallback = lib.hm.dag.entryAfter [ "writeBoundary" ] ''
          theme_dir="$HOME/.config/ghostty/themes"
          theme_file="$theme_dir/noctalia"

          ${pkgs.coreutils}/bin/mkdir -p "$theme_dir"

          if [ ! -e "$theme_file" ]; then
            cat >"$theme_file" <<'THEME'
    background = #${config.lib.stylix.colors.base00}
    foreground = #${config.lib.stylix.colors.base05}
    cursor-color = #${config.lib.stylix.colors.base0D}
    cursor-text = #${config.lib.stylix.colors.base00}
    selection-background = #${config.lib.stylix.colors.base0D}
    selection-foreground = #${config.lib.stylix.colors.base00}
    palette = 0=#${config.lib.stylix.colors.base00}
    palette = 1=#${config.lib.stylix.colors.base08}
    palette = 2=#${config.lib.stylix.colors.base0B}
    palette = 3=#${config.lib.stylix.colors.base0A}
    palette = 4=#${config.lib.stylix.colors.base0D}
    palette = 5=#${config.lib.stylix.colors.base0E}
    palette = 6=#${config.lib.stylix.colors.base0C}
    palette = 7=#${config.lib.stylix.colors.base05}
    palette = 8=#${config.lib.stylix.colors.base03}
    palette = 9=#${config.lib.stylix.colors.base08}
    palette = 10=#${config.lib.stylix.colors.base0B}
    palette = 11=#${config.lib.stylix.colors.base0A}
    palette = 12=#${config.lib.stylix.colors.base0D}
    palette = 13=#${config.lib.stylix.colors.base0E}
    palette = 14=#${config.lib.stylix.colors.base0C}
    palette = 15=#${config.lib.stylix.colors.base07}
    THEME
          fi
  '';

  programs.ghostty = {
    enable = true;
    package = pkgs.ghostty;
    systemd.enable = true;
    enableZshIntegration = settings.shell == "zsh";

    settings = {
      command = "${ghosttyShell}";

      theme = "noctalia";

      font-family = settings.themeDetails.font;
      font-size = settings.themeDetails.fontSize;

      window-padding-x = 15;
      window-padding-y = 15;
      window-padding-balance = true;
      window-decoration = "none";

      cursor-style = "underline";
      cursor-style-blink = false;

      shell-integration = "detect";
      shell-integration-features = "no-cursor,title";

      confirm-close-surface = true;

      clipboard-read = "ask";
      clipboard-write = "allow";
      clipboard-paste-protection = true;
      clipboard-paste-bracketed-safe = false;
      copy-on-select = false;

      title-report = false;
      scrollback-limit = 10485760;
    };
  };
}
