{
  config,
  lib,
  pkgs,
  gjallarctlPackage,
  settings,
  gjallarSecureBootArtifactVerifier,
  gjallarSecureBootOwnershipVerifier,
  ...
}:
let
  luksMappings = builtins.attrValues config.boot.initrd.luks.devices;
  luksDevice = if builtins.length luksMappings == 1 then (builtins.head luksMappings).device else "";
in
{
  assertions = lib.optionals settings.luksTpm2Enable [
    {
      assertion = settings.secureBootEnable;
      message = "GjallarOS TPM2 unlock requires Secure Boot.";
    }
    {
      assertion = builtins.length luksMappings == 1 && lib.hasPrefix "/dev/disk/by-uuid/" luksDevice;
      message = "GjallarOS TPM2 enrollment requires exactly one LUKS device addressed by /dev/disk/by-uuid/.";
    }
  ];

  systemd.services.gjallar-tpm2-enroll =
    lib.mkIf (settings.secureBootEnable && settings.luksTpm2Enable)
      {
        description = "Enroll GjallarOS measured-boot TPM2 LUKS token";
        wantedBy = [ "multi-user.target" ];
        after = [
          "gjallar-secure-boot-finalize.service"
          "systemd-pcrlock-make-policy.service"
        ];
        requires = [ "systemd-pcrlock-make-policy.service" ];
        unitConfig = {
          ConditionSecurity = "uefi-secureboot";
          ConditionPathExists = [
            "!/var/lib/gjallarOS/tpm2-enrollment-complete"
            "/var/lib/systemd/pcrlock.json"
            "/var/lib/gjallarOS/secure-boot/ownership.json"
          ];
        };
        serviceConfig = {
          Type = "oneshot";
          UMask = "0077";
        };
        path = [
          pkgs.cryptsetup
          gjallarctlPackage
          pkgs.systemd
          pkgs.coreutils
          pkgs.gnugrep
          gjallarSecureBootArtifactVerifier
          gjallarSecureBootOwnershipVerifier
        ];
        script = ''
          set -euo pipefail
          device=${lib.escapeShellArg luksDevice}
          state=/var/lib/gjallarOS
          keyfile="$(mktemp /run/gjallar-luks-key.XXXXXX)"
          trap 'rm -f -- "$keyfile"' EXIT

          [ ! -e "$state/secure-boot-enable-required" ] || exit 0
          gjallar-verify-secure-boot-artifacts
          gjallar-verify-secure-boot-ownership enrolled

          systemd-ask-password --timeout=0 \
            "GjallarOS: enter the human LUKS recovery passphrase to enroll measured-boot TPM2 unlock" >"$keyfile"
          chmod 0600 "$keyfile"
          cryptsetup open --test-passphrase --type luks "$device" --key-file "$keyfile"

          # Fresh installs must not inherit an unidentified TPM token. Refuse an
          # ambiguous state instead of deleting any unknown slot.
          before="$(cryptsetup luksDump --dump-json-metadata "$device")"
          if printf '%s' "$before" | grep -q 'systemd-tpm2'; then
            printf '%s\n' 'ERROR: an existing TPM2 token needs explicit audited migration; no slot was changed.' >&2
            exit 1
          fi

          systemd-cryptenroll --unlock-key-file="$keyfile" --tpm2-device=auto \
            --tpm2-pcrlock=/var/lib/systemd/pcrlock.json "$device"

          mkdir -p "$state/luks"
          cryptsetup luksDump --dump-json-metadata "$device" >"$state/luks/metadata.json.tmp"
          token_id="$(
            gjallarctl installer tpm2-metadata-token-id \
              "$state/luks/metadata.json.tmp"
          )"
          cryptsetup open --test-passphrase --token-only --token-id "$token_id" "$device"
          gjallarctl installer tpm2-write-keyslot-record \
            "$state/luks/metadata.json.tmp" \
            "$state/luks/keyslots.json" \
            "$device" \
            /var/lib/systemd/pcrlock.json
          rm -f -- "$state/luks/metadata.json.tmp"
          chmod 0600 "$state/luks/keyslots.json"
          install -m 0600 /dev/null "$state/tpm2-enrollment-complete"
          printf '%s\n' 'GjallarOS measured-boot TPM2 enrollment completed; human recovery keyslots were preserved.'
        '';
      };

}
