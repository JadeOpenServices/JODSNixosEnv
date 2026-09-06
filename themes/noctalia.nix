{ pkgs, ... }:
{
  # Noctalia owns the theme. These defaults preserve the previous desktop's
  # visual character while allowing CCS changes to take over at runtime.
  themeName = "rose-pine";
  avatar = ../non-nix/wallpapers/avatar.png;
  wallpaper = {
    left = ../non-nix/wallpapers/evening-sky.png;
    center = ../non-nix/wallpapers/evening-sky.png;
    right = ../non-nix/wallpapers/evening-sky.png;
  };
  override = {
    base00 = "11111b";
    base01 = "181825";
    base02 = "313244";
    base03 = "45475a";
    base04 = "585b70";
    base05 = "cdd6f4";
    base06 = "f5e0dc";
    base07 = "b4befe";
    base08 = "f38ba8";
    base09 = "fab387";
    base0A = "f9e2af";
    base0B = "a6e3a1";
    base0C = "94e2d5";
    base0D = "89b4fa";
    base0E = "cba6f7";
    base0F = "f2cdcd";
  };

  btopTheme = null;
  shell = "noctalia";

  opacity = 0.8;
  rounding = 25;
  shadow = false;
  bordersPlusPlus = false;

  font = "FiraCode Nerd Font";
  fontPkg = pkgs.nerd-fonts.fira-code;
  fontSize = 13;

  icons = "Papirus";
  iconsPkg = pkgs.papirus-icon-theme;
}
