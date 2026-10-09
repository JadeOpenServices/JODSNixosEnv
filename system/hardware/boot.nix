{
  config,
  lib,
  pkgs,
  settings,
  ...
}:
{
  boot.loader.efi.canTouchEfiVariables = true;
  boot.loader.systemd-boot.enable = true;
  # The boot menu editor lets anyone at the keyboard append init=/bin/sh and
  # get a root shell, which also bypasses TPM-unlocked disk encryption.
  boot.loader.systemd-boot.editor = false;

  # Lanzaboote's UKIs outlive a switch back from Secure Boot: this builder
  # never looks in EFI/Linux, and once their closures are collected they
  # boot into a kernel panic (live host, 2026-10-05).
  boot.loader.systemd-boot.extraInstallCommands = ''
    ${pkgs.coreutils}/bin/rm -f ${lib.escapeShellArg config.boot.loader.efi.efiSysMountPoint}/EFI/Linux/nixos-generation-*.efi
  '';

  # nixos-generate-config mounts the ESP fmask=0022/dmask=0022, so every
  # user can read the boot loader random seed (bootctl warns about it) and
  # the boot files. Appended so they win over the generated masks; vfat
  # takes the last value.
  fileSystems.${config.boot.loader.efi.efiSysMountPoint}.options = lib.mkAfter [
    "fmask=0077"
    "dmask=0077"
  ];

  boot.loader.timeout = if settings.debugFunctions then 5 else 0;
}
