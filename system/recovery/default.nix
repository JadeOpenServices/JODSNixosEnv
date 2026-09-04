{
  config,
  lib,
  pkgs,
  settings,
  ...
}:
let
  recovery = pkgs.writeShellApplication {
    name = "gjallar-recover";
    runtimeInputs = [
      config.nix.package
      pkgs.nixos-rebuild
      pkgs.systemd
    ];
    text = ''
      set -euo pipefail

      case "''${1:-help}" in
        list)
          nix-env --list-generations --profile /nix/var/nix/profiles/system
          ;;
        rollback)
          printf 'Roll back to the previous NixOS generation? [y/N] '
          read -r answer
          case "$answer" in
            y|Y|yes|YES|Yes) sudo nixos-rebuild switch --rollback ;;
            *) exit 1 ;;
          esac
          ;;
        reboot)
          sudo systemctl reboot
          ;;
        *)
          printf '%s\n' \
            'GjallarOS recovery' \
            '  gjallar-recover list      List system generations' \
            '  gjallar-recover rollback  Activate the previous generation' \
            '  gjallar-recover reboot    Reboot the machine'
          ;;
      esac
    '';
  };
in
lib.mkMerge [
  (lib.mkIf settings.recoveryEnable {
    # Keep the boot menu reachable whenever a dedicated recovery entry exists.
    boot.loader.timeout = lib.mkForce 5;

    specialisation.gjallar-recovery.configuration = {
      system.nixos.tags = [ "recovery" ];
      boot.kernelParams = [ "systemd.unit=multi-user.target" ];
      environment.systemPackages = [ recovery ];
      services.openssh.enable = lib.mkForce false;

      environment.etc."gjallar/recovery".text = ''
        This is the local GjallarOS recovery boot entry.
        Sign in at the console and run: gjallar-recover
      '';
    };
  })

  # Reserved local policy input for JODS. This records intent only; strong
  # stolen-device locking additionally requires Secure Boot, measured boot,
  # remote attestation, and revocable disk-key release.
  (lib.mkIf (settings.recoveryEnable || settings.jodsPrebootLockEnable) {
    environment.etc."jods/preboot-policy".text = ''
      recovery=${if settings.recoveryEnable then "enabled" else "disabled"}
      lock=${if settings.jodsPrebootLockEnable then "requested" else "disabled"}
    '';
  })
]
