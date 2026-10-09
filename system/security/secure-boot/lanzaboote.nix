{
  config,
  lib,
  pkgs,
  settings,
  ...
}:
let
  # systemd-pcrlock supports at most 8 alternatives per PCR, and PCR 4
  # measures every installed UKI: one per generation plus one per
  # specialisation (trusted recovery, recovery-storage maintenance). With
  # more, every bootloader install fails ("Failed to calculate super PCR
  # policy: Argument list too long"), e2e-full, 2026-09-29.
  pcrlockAlternatives = 8;
  ukisPerGeneration = 1 + builtins.length (builtins.attrNames config.specialisation);
  esp = lib.escapeShellArg config.boot.loader.efi.efiSysMountPoint;
in
{
  assertions = lib.optional settings.luksTpm2Enable {
    assertion = ukisPerGeneration <= pcrlockAlternatives;
    message = "Measured boot supports at most ${toString pcrlockAlternatives} UKIs per generation; ${toString ukisPerGeneration} are configured.";
  };

  boot.loader.grub.enable = lib.mkIf settings.secureBootEnable (lib.mkForce false);
  boot.loader.systemd-boot.enable = lib.mkIf settings.secureBootEnable (lib.mkForce false);

  boot.lanzaboote = lib.mkIf settings.secureBootEnable {
    enable = true;
    pkiBundle = "/var/lib/sbctl";
    # With recovery, older generations are chosen there (gjallar-recover
    # rollback), not in the boot menu.
    configurationLimit =
      if settings.recoveryEnable or false then
        1
      else if settings.luksTpm2Enable then
        lib.max 1 (pcrlockAlternatives / ukisPerGeneration)
      else
        8;

    measuredBoot = lib.mkIf settings.luksTpm2Enable {
      enable = true;

      pcrs = [
        0
        4
        7
      ];
    };
  };

  # systemd-boot's entries outlive a switch to Secure Boot, and lanzaboote
  # already removed the kernels they load.
  #
  # lzbt syncs the ESP before its garbage collection and before make-policy
  # writes the pcrlock credential. On vfat a rename over an existing file
  # reaches the disk through the file's inode, which make-policy's directory
  # fsync does not write. A power loss within ~30 s left the old directory
  # entry pointing at freed clusters; fsck cut the credential to 0 bytes and
  # TPM unlock failed with "Encrypted file too short" (e2e-fw13, 2026-10-10,
  # after rollback). Sync last, as the systemd-boot builder does.
  system.build.installBootLoader = lib.mkIf settings.secureBootEnable (
    lib.mkForce (
      pkgs.writeShellScript "install-lanzaboote" ''
        set -euo pipefail
        ${config.boot.loader.external.installHook} "$@"
        ${pkgs.coreutils}/bin/rm -f ${esp}/loader/entries/nixos*-generation-*.conf
        ${pkgs.coreutils}/bin/sync --file-system ${esp}
      ''
    )
  );
}
