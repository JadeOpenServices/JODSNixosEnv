{ pkgs, ... }:
{
  themeName = "black-metal-immortal";
  wallpaper = ../non-nix/wallpapers/black.png;
  override = null;

  # Override stylix theme of btop.
  btopTheme = "nord";

  opacity = 1.0;
  rounding = 25;
  shadow = true;
  bordersPlusPlus = false;
  font = "FiraCode Nerd Font"; # Selected font
  fontPkg = (pkgs.nerd-fonts.fira-code);
  fontSize = 13; # Font size

  icons = "Papirus";
  iconsPkg = pkgs.papirus-icon-theme;
}
