{ pkgs, ... }:
{
  # Native, offline-first KDBX password manager.
  #
  # KeePassXC owns its mutable preferences. GjallarOS deliberately does not
  # enable browser access, Secret Service, SSH-agent integration, Auto-Type,
  # autostart, or any other ambient secret-access path.
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
