{
  lib,
  settings,
  ...
}:
{
  boot.loader.grub.enable = lib.mkIf settings.secureBootEnable (lib.mkForce false);
  boot.loader.systemd-boot.enable = lib.mkIf settings.secureBootEnable (lib.mkForce false);

  boot.lanzaboote = lib.mkIf settings.secureBootEnable {
    enable = true;
    pkiBundle = "/var/lib/sbctl";
    configurationLimit = 8;

    measuredBoot = lib.mkIf settings.luksTpm2Enable {
      enable = true;

      # PCR 4 covers the complete Lanzaboote boot chain; PCR 7 covers the
      # Secure Boot policy. PCR 0 also binds firmware code. Lanzaboote keeps
      # the pcrlock policy current across generation updates.
      pcrs = [ 0 4 7 ];
    };
  };
}
