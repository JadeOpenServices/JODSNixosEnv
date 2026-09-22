{
  lib,
  pkgs,
  settings,
  ...
}:

lib.mkIf settings.containersEnable {
  environment.systemPackages = with pkgs; [
    podman-compose
    distrobox
    podman-tui
  ];

  virtualisation.podman = {
    enable = true;
    dockerCompat = true;
    defaultNetwork.settings.dns_enabled = true;
  };

  virtualisation.containers.registries.search = [ "docker.io" ];
}
