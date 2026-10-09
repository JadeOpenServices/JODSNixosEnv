{ lib, settings, ... }:
# Its own module: next to programs.${settings.shell} in system/default.nix, a
# static programs.zsh is a duplicate attribute whenever the shell is zsh.
lib.mkIf (settings.shell == "zsh") {
  # Home Manager runs compinit in the user's .zshrc. A second run from
  # /etc/zshrc repeated the completion audit at every shell start.
  programs.zsh.enableGlobalCompInit = false;
}
