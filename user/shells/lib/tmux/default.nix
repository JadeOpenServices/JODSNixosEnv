{ pkgs, settings, ... }:
let
  themePath = ./themes/${settings.theme}.nix;
  theme = if builtins.pathExists themePath then import themePath { inherit pkgs; } else [ ];
in
{
  programs.tmux = {
    enable = true;
    prefix = "C-s";
    terminal = "tmux-256color";
    escapeTime = 0;
    mouse = true;
    historyLimit = 10000;
    disableConfirmationPrompt = true;
    plugins =
      with pkgs.tmuxPlugins;
      [
        {
          plugin = resurrect;
          extraConfig = ''
            set -g @resurrect-capture-pane-contents 'on'
            set -g @resurrect-processes 'lazygit'
          '';
        }
      ]
      ++ theme;
    extraConfig = ''
      set -Fg 'status-format[1]' '#{status-format[0]}'
      set -g 'status-format[0]' ""
      set -g status 2

      ####################
      ####################

      unbind r
      bind r source-file ~/.config/tmux/tmux.conf

      bind-key -n C-Left previous-window
      bind-key -n C-Right next-window
    '';
  };
}
