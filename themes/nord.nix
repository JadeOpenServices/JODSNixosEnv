{ pkgs, ... }:
{
  themeName = "nord";
  override = {
    base02 = "#445060";
    base05 = "#fffcf0";
  };

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
