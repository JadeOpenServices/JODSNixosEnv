{ ... }:

let
  GiB = 1024 * 1024 * 1024;
in
{
  nix.gc = {
    automatic = true;
    dates = "weekly";
    persistent = true;
    randomizedDelaySec = "2h";
    options = "--delete-older-than 14d";
  };

  nix.settings = {
    min-free = 10 * GiB;
    max-free = 30 * GiB;

    auto-optimise-store = false;

    keep-outputs = false;

    keep-derivations = true;
  };

  nix.optimise = {
    automatic = true;
    dates = [ "weekly" ];
  };

  boot.tmp.cleanOnBoot = true;

  services.journald.extraConfig = ''
    SystemMaxUse=1G
    RuntimeMaxUse=256M
    MaxRetentionSec=14day
  '';

  # Builds yield to the desktop. With default weights a rebuild's 16 build
  # jobs got as much CPU as the whole user session, and a new VSCodium
  # window took 14s to appear (2026-10-08). Weights only matter under
  # contention, so idle builds still use every core.
  nix.daemonCPUSchedPolicy = "batch";
  nix.daemonIOSchedClass = "best-effort";
  nix.daemonIOSchedPriority = 7;
  systemd.services.nix-daemon.serviceConfig = {
    CPUWeight = 20;
    IOWeight = 20;
  };

  systemd.services.nix-gc.serviceConfig = {
    Nice = 19;
    IOSchedulingClass = "idle";
  };

  systemd.services.nix-optimise.serviceConfig = {
    Nice = 19;
    IOSchedulingClass = "idle";
  };
}
