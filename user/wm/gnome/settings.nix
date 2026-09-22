{
  inputs,
  config,
  pkgs,
  ...
}:

{
  dconf.settings = {
    "/org/gnome/desktop/interface" = {
    };
    "/org/gnome/mutter" = {
      center-new-windows = true;
    };
    "/org/gnome/mutter" = {
      experimental-features = "['scale-monitor-framebuffer']";
    };
  };
}
