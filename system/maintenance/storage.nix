{ ... }:

let
  GiB = 1024 * 1024 * 1024;
in
{
  # Keep useful rollback history, but do not retain generations forever.
  nix.gc = {
    automatic = true;
    dates = "weekly";
    persistent = true;
    randomizedDelaySec = "2h";
    options = "--delete-older-than 14d";
  };

  # Emergency guardrail for builds. If free space drops below 10 GiB,
  # collect unreferenced store paths until 30 GiB is available or there
  # is no more garbage.
  nix.settings = {
    min-free = 10 * GiB;
    max-free = 30 * GiB;

    # Do not add optimisation work to every build.
    auto-optimise-store = false;

    # Outputs not otherwise reachable from a GC root are disposable.
    keep-outputs = false;

    # Keep derivations for traceability of live store paths.
    keep-derivations = true;
  };

  # Deduplicate identical store files out of the rebuild critical path.
  nix.optimise = {
    automatic = true;
    dates = [ "weekly" ];
  };

  # Stale interrupted-build scratch data must not survive indefinitely.
  boot.tmp.cleanOnBoot = true;

  # Prevent logging faults/restart loops from consuming arbitrary disk space.
  services.journald.extraConfig = ''
    SystemMaxUse=1G
    RuntimeMaxUse=256M
    MaxRetentionSec=14day
  '';

  # Housekeeping must yield to interactive work and builds.
  systemd.services.nix-gc.serviceConfig = {
    Nice = 19;
    IOSchedulingClass = "idle";
  };

  systemd.services.nix-optimise.serviceConfig = {
    Nice = 19;
    IOSchedulingClass = "idle";
  };
}
