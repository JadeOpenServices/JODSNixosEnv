{ pkgs, ... }:
{
  home.packages = [
    pkgs.keepassxc
  ];

  xdg.mimeApps = {
    enable = true;

    defaultApplications = {
      "application/x-keepass2" = "org.keepassxc.KeePassXC.desktop";
    };
  };
}
