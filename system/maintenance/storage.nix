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

  systemd.services.nix-gc.serviceConfig = {
    Nice = 19;
    IOSchedulingClass = "idle";
  };

  systemd.services.nix-optimise.serviceConfig = {
    Nice = 19;
    IOSchedulingClass = "idle";
  };
}
