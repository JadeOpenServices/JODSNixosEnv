{
  config,
  lib,
  pkgs,
  settings,
  sourceRevision,
  ...
}:
let
  tools = import ./tools.nix {
    inherit config pkgs sourceRevision;
  };
  inherit (tools) gjallarctl recovery recoveryExecutor;

  maintenanceCandidates = lib.concatStringsSep "\n" (
    lib.mapAttrsToList (
      name: value:
      let
        device = value.device or "";
      in
      lib.optionalString (device != "") ''
        try_candidate ${lib.escapeShellArg name} ${lib.escapeShellArg device}
      ''
    ) config.boot.initrd.luks.devices
  );

  maintenanceNext = pkgs.writeShellApplication {
    name = "gjallar-recovery-maintenance-next";

    runtimeInputs = [
      pkgs.coreutils
      pkgs.python3
      pkgs.systemd
    ];

    text = ''
            set -euo pipefail

            if [ "$(${pkgs.coreutils}/bin/id -u)" -ne 0 ]; then
              exec sudo "$0" "$@"
            fi

            root="''${1:-/}"

            entry="$(
              bootctl --root="$root" list --json=short |
                ${pkgs.python3}/bin/python3 -c '
      import json
      import sys

      entries = json.load(sys.stdin)
      matches = []

      for entry in entries:
          haystack = " ".join(
              str(entry.get(field, ""))
              for field in ("id", "title", "path")
          )

          if "gjallar-recovery-maintenance" in haystack:
              entry_id = entry.get("id")

              if entry_id:
                  matches.append(entry_id)

      if len(matches) != 1:
          raise SystemExit(
              "expected exactly one GjallarOS recovery-maintenance "
              f"boot entry, found {len(matches)}"
          )

      print(matches[0])
      '
            )"

            test -n "$entry"

            bootctl --root="$root" set-oneshot "$entry"

            printf '%s\n' \
              "PASS: armed one-shot GjallarOS recovery-storage maintenance boot" \
              "PASS: normal default boot entry was not changed" \
              "Rebooting into maintenance..."

            systemctl reboot
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
        recoveryExecutor
        gjallarctl
        pkgs.dosfstools
        pkgs.gptfdisk
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
    }
    // lib.optionalAttrs settings.endpointManagedDevice {
      services.jods-mdm-agent.recoveryExecutorEnable = true;
    };
  })

  (lib.mkIf settings.recoveryEnable {
    environment.systemPackages = [
      maintenanceNext
    ];

    specialisation.gjallar-recovery-maintenance.configuration = {
      system.nixos.tags = [
        "recovery-storage-maintenance"
      ];

      boot.loader.timeout = lib.mkForce 5;

      boot.initrd.systemd.services.gjallar-recovery-maintenance = {
        description = "Provision GjallarOS recovery storage before mounting the installed root";

        before = [
          "sysroot.mount"
          "initrd-root-fs.target"
        ];

        requiredBy = [
          "sysroot.mount"
          "initrd-root-fs.target"
        ];

        unitConfig = {
          DefaultDependencies = false;
          OnFailure = "emergency.target";
        };

        serviceConfig = {
          Type = "oneshot";
          UMask = "0077";
          StandardOutput = "journal+console";
          StandardError = "journal+console";
        };

        path = [
          pkgs.btrfs-progs
          pkgs.coreutils
          pkgs.cryptsetup
          pkgs.gptfdisk
          pkgs.systemd
          pkgs.util-linux
          gjallarctl
        ];

        script = ''
          set -euo pipefail
          export LC_ALL=C

          keyfile=/run/gjallar-recovery-maintenance.key

          cleanup_key() {
            rm -f -- "$keyfile"
          }

          trap cleanup_key EXIT

          printf '%s\n' \
            "GjallarOS recovery-storage maintenance" \
            "The installed root is not mounted as /." \
            "A human LUKS credential is required before /mnt is exposed."

          systemd-ask-password --timeout=0 \
            "GjallarOS: enter the LUKS recovery credential for storage maintenance" \
            > "$keyfile"

          chmod 0600 "$keyfile"

          selected_device=""
          selected_mapping=""
          selected_name=""

          try_candidate() {
            name="$1"
            device="$2"
            mapper="/dev/mapper/$name"
            opened_here=false

            [ -e "$device" ] || return 0

            if [ ! -e "$mapper" ]; then
              if ! cryptsetup open \
                --type luks \
                --key-file "$keyfile" \
                "$device" \
                "$name"
              then
                return 0
              fi

              opened_here=true
            fi

            filesystem="$(
              blkid -o value -s TYPE "$mapper" 2>/dev/null || true
            )"

            if [ "$filesystem" != btrfs ]; then
              if [ "$opened_here" = true ]; then
                cryptsetup close "$name"
              fi

              return 0
            fi

            if [ -n "$selected_device" ]; then
              printf '%s\n' \
                "ERROR: more than one configured LUKS mapping contains Btrfs." \
                "No storage changes were made." >&2

              if [ "$opened_here" = true ]; then
                cryptsetup close "$name"
              fi

              return 77
            fi

            selected_device="$device"
            selected_mapping="$mapper"
            selected_name="$name"
          }

          ${maintenanceCandidates}

          if [ -z "$selected_device" ] ||
             [ -z "$selected_mapping" ] ||
             [ -z "$selected_name" ]
          then
            printf '%s\n' \
              "ERROR: no configured encrypted Btrfs root accepted the supplied credential." \
              "No storage changes were made." >&2

            systemctl --no-block emergency
            false
          fi

          printf '%s\n' \
            "PASS: authenticated encrypted Btrfs root selected" \
            "STAGE: mounting root at /mnt and provisioning recovery storage"

          if ! gjallar-recovery-maintenance \
            --root-partition "$selected_device" \
            --mapping "$selected_mapping" \
            < "$keyfile"
          then
            printf '%s\n' \
              "ERROR: recovery-storage maintenance failed." \
              "The normal installed root was not booted." >&2

            systemctl --no-block emergency
            false
          fi

          rm -f -- "$keyfile"
          trap - EXIT

          sync

          printf '%s\n' \
            "PASS: exact 12 GiB JODS-RECOVERY provisioned" \
            "PASS: installed root was serviced only at /mnt" \
            "PASS: maintenance credential removed" \
            "Rebooting to the normal GjallarOS boot path..."

          systemctl reboot --force --force
        '';
      };
    };
  })

  (lib.mkIf (settings.recoveryEnable || settings.jodsPrebootLockEnable) {
    environment.etc."jods/preboot-policy".text = ''
      version=1
      recovery=${if settings.recoveryEnable then "signed-independent-xbootldr" else "disabled"}
      root_unlock=human-recovery-credential-only
      external_media_tpm_unlock=forbidden
      jods_lock=${if settings.jodsPrebootLockEnable then "measured-boot-required" else "disabled"}
      modes=repair,reinstall,fresh
      default_mode=repair
    '';
  })
]
