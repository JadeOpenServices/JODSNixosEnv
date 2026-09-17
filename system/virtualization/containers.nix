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

  # Container tooling is rootless. Docker-compatible commands are backed by
  # the calling user's Podman storage and user namespace.
  virtualisation.podman = {
    enable = true;
    dockerCompat = true;
    defaultNetwork.settings.dns_enabled = true;
  };

  # Resolve OCI short names deterministically. Project-owned Containerfiles
  # should still use fully qualified image references.
  virtualisation.containers.registries.search = [ "docker.io" ];
}
