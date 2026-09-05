{
  config,
  lib,
  pkgs,
  settings,
  ...
}:
let
  gjallarctl = pkgs.callPackage ../../pkgs/gjallarctl { };
  recovery = pkgs.writeShellApplication {
    name = "gjallar-recover";
    runtimeInputs = with pkgs; [
      config.nix.package
      cryptsetup
      git
      nixos-install-tools
      sbctl
      systemd
      tpm2-tools
      util-linux
      gjallarctl
    ];
    text = ''
      set -euo pipefail

      usage() {
        printf '%s\n' \
          'GjallarOS trusted recovery' \
          '  gjallar-recover audit' \
          '  gjallar-recover unlock DEVICE NAME' \
          '  gjallar-recover mount DEVICE MOUNTPOINT' \
          '  gjallar-recover generations ROOT' \
          '  gjallar-recover repair-boot ROOT' \
          '  gjallar-recover rebuild ROOT FLAKE#HOST' \
          '  gjallar-recover jods {repair|reinstall|fresh}'
      }

      require_root() {
        if [ "$(id -u)" -ne 0 ]; then
          printf '%s\n' 'ERROR: run this operation as root.' >&2
          exit 1
        fi
      }

      confirm_phrase() {
        expected="$1"
        printf 'Type %s to continue: ' "$expected"
        read -r answer
        [ "$answer" = "$expected" ] || {
          printf '%s\n' 'Cancelled.' >&2
          exit 1
        }
      }

      case "''${1:-help}" in
        audit)
          printf '%s\n' '=== Secure Boot ==='
          sbctl status || true
          printf '%s\n' '=== TPM ==='
          systemd-analyze has-tpm2 || true
          tpm2_getcap properties-fixed || true
          printf '%s\n' '=== disks ==='
          lsblk -o NAME,PATH,TYPE,FSTYPE,UUID,PARTUUID,MOUNTPOINTS
          ;;
        unlock)
          require_root
          device="''${2:?DEVICE required}"
          name="''${3:?mapping NAME required}"
          cryptsetup isLuks "$device"
          systemd-ask-password 'GjallarOS LUKS recovery passphrase:' |
            cryptsetup open --type luks --key-file=- "$device" "$name"
          ;;
        mount)
          require_root
          device="''${2:?DEVICE required}"
          target="''${3:?MOUNTPOINT required}"
          mkdir -p "$target"
          mount "$device" "$target"
          ;;
        generations)
          root="''${2:?installed ROOT mountpoint required}"
          nix-env --list-generations --profile "$root/nix/var/nix/profiles/system"
          ;;
        repair-boot)
          require_root
          root="''${2:?installed ROOT mountpoint required}"
          [ -e "$root/etc/NIXOS" ] || {
            printf '%s\n' 'ERROR: target is not a mounted NixOS installation.' >&2
            exit 1
          }
          confirm_phrase REPAIR-BOOT
          NIXOS_INSTALL_BOOTLOADER=1 nixos-enter --root "$root" -- \
            /run/current-system/bin/switch-to-configuration boot
          ;;
        rebuild)
          require_root
          root="''${2:?installed ROOT mountpoint required}"
          flake="''${3:?FLAKE#HOST required}"
          [ -e "$root/etc/NIXOS" ] || {
            printf '%s\n' 'ERROR: target is not a mounted NixOS installation.' >&2
            exit 1
          }
          confirm_phrase REBUILD
          nixos-enter --root "$root" -- nixos-rebuild boot --flake "$flake"
          ;;
        jods)
          mode="''${2:-}"
          case "$mode" in
            repair)
              printf '%s\n' \
                'JODS mode: REPAIR EXISTING INSTALLATION' \
                'Disk formatting is forbidden. Unlock with a human recovery credential, then use repair-boot or rebuild.'
              ;;
            reinstall)
              printf '%s\n' \
                'JODS mode: REINSTALL EXISTING INSTALLATION' \
                'Existing partitions and encrypted data must be detected before any installer runs.'
              confirm_phrase REINSTALL-EXISTING
              gjallar-installer --accept-existing
              ;;
            fresh)
              printf '%s\n' \
                'DANGER: JODS FRESH INSTALL MODE' \
                'Select the target disk in the installer and confirm its destructive plan.'
              confirm_phrase ERASE-FOR-FRESH-INSTALL
              gjallar-installer
              ;;
            *)
              printf '%s\n' 'Usage: gjallar-recover jods {repair|reinstall|fresh}' >&2
              exit 2
              ;;
          esac
          ;;
        *) usage ;;
      esac
    '';
  };
in
lib.mkMerge [
  {
    assertions = lib.optional settings.jodsPrebootLockEnable {
      assertion =
        settings.endpointManagedDevice
        && settings.recoveryEnable
        && settings.secureBootEnable
        && settings.luksTpm2Enable;
      message = "JODS preboot locking requires explicit endpoint management enrollment, recoveryEnable, secureBootEnable, and luksTpm2Enable.";
    };
  }

  (lib.mkIf settings.recoveryEnable {
    boot.loader.timeout = lib.mkForce 5;

    specialisation.gjallar-recovery.configuration = {
      system.nixos.tags = [ "trusted-recovery" ];
      boot.kernelParams = [ "systemd.unit=multi-user.target" ];
      environment.systemPackages = [
        recovery
        gjallarctl
        pkgs.dosfstools
        pkgs.gdisk
        pkgs.parted
        pkgs.xorriso
      ];
      services.openssh.enable = lib.mkForce false;
      networking.firewall.enable = lib.mkForce true;

      environment.etc."gjallar/recovery".text = ''
        Trusted GjallarOS recovery entry.
        Sign in locally and run: sudo gjallar-recover audit
        Root disks require a verified human LUKS recovery credential.
      '';
    };
  })

  (lib.mkIf (settings.recoveryEnable || settings.jodsPrebootLockEnable) {
    environment.etc."jods/preboot-policy".text = ''
      version=1
      recovery=${if settings.recoveryEnable then "signed-local-entry" else "disabled"}
      root_unlock=human-recovery-credential-only
      external_media_tpm_unlock=forbidden
      jods_lock=${if settings.jodsPrebootLockEnable then "measured-boot-required" else "disabled"}
      modes=repair,reinstall,fresh
      default_mode=repair
    '';
  })
]
