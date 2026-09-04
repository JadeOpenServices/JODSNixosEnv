{
  lib,
  pkgs,
  settings,
  ...
}:
if settings.frameworkEnable then
  let
    profile = builtins.fromJSON (builtins.readFile (./profiles + "/${settings.frameworkModel}.json"));
    configJson = builtins.toJSON {
      defaultStrategy = profile.defaultStrategy;
      strategyOnDischarging = profile.strategyOnDischarging;
      strategies = profile.strategies;
    };

    secureBootArtifactVerifier = pkgs.writeShellApplication {
      name = "gjallar-verify-secure-boot-artifacts";
      runtimeInputs = [
        pkgs.sbctl
      pkgs.util-linux
        pkgs.coreutils
        pkgs.gnugrep
      ];
      text = ''
        set -u
        export LC_ALL=C

        if output="$(sbctl verify 2>&1)"; then
  rc=0
else
  rc=$?
fi

        printf '%s\n' "$output"

        unexpected="$(
          printf '%s\n' "$output" |
            grep -F 'is not signed' |
            grep -Ev '/boot/EFI/nixos/kernel-[^ ]+\.efi is not signed$' ||
            true
        )"

        if [ -n "$unexpected" ]; then
          printf '%s\n' \
            'ERROR: unexpected unsigned EFI artifact:' \
            "$unexpected" >&2
          exit 1
        fi

        if [ "$rc" -ne 0 ]; then
          allowed="$(
            printf '%s\n' "$output" |
              grep -E '/boot/EFI/nixos/kernel-[^ ]+\.efi is not signed$' ||
              true
          )"

          if [ -z "$allowed" ]; then
            printf '%s\n' \
              'ERROR: sbctl verify failed for an unexpected reason.' >&2
            exit "$rc"
          fi
        fi

        printf '%s\n' \
          'GjallarOS boot artifact verification passed.' \
          'The external Lanzaboote kernel remains byte-identical and is hash-verified by the signed generation stub.'
      '';
    };
    secureBootOwnershipVerifier = pkgs.writeShellApplication {
      name = "gjallar-verify-secure-boot-ownership";
      runtimeInputs = [
        pkgs.python3
      ];

      text = ''
        set -euo pipefail

        python3 - "''${1:-}" <<'PYEOF'
import base64
import glob
import hashlib
import json
import re
import sys
from pathlib import Path

expected_stage = sys.argv[1] or None
ownership = Path("/var/lib/gjallarOS/secure-boot/ownership.json")

if not ownership.is_file():
    raise SystemExit(
        "ERROR: GjallarOS Secure Boot ownership metadata is missing."
    )

doc = json.loads(ownership.read_text())

if doc.get("schema") != 1:
    raise SystemExit(
        f"ERROR: unsupported ownership schema: {doc.get('schema')!r}"
    )

if doc.get("manager") != "gjallarOS":
    raise SystemExit(
        f"ERROR: unexpected Secure Boot manager: {doc.get('manager')!r}"
    )

if expected_stage is not None and doc.get("stage") != expected_stage:
    raise SystemExit(
        "ERROR: ownership stage is "
        f"{doc.get('stage')!r}; expected {expected_stage!r}."
    )

guid_path = Path("/var/lib/sbctl/GUID")
if not guid_path.is_file():
    raise SystemExit("ERROR: /var/lib/sbctl/GUID is missing.")

guid = guid_path.read_text().strip()

if doc.get("sbctlOwnerGuid") != guid:
    raise SystemExit(
        "ERROR: ownership record GUID does not match the current sbctl GUID."
    )

certs = {
    "PK": (
        Path("/var/lib/sbctl/keys/PK/PK.pem"),
        "pkSha256",
    ),
    "KEK": (
        Path("/var/lib/sbctl/keys/KEK/KEK.pem"),
        "kekSha256",
    ),
    "db": (
        Path("/var/lib/sbctl/keys/db/db.pem"),
        "dbSha256",
    ),
}

def pem_der(path):
    text = path.read_text()
    match = re.search(
        r"-----BEGIN CERTIFICATE-----\s*(.*?)\s*-----END CERTIFICATE-----",
        text,
        re.S,
    )
    if not match:
        raise SystemExit(
            f"ERROR: {path} does not contain an X.509 certificate."
        )

    payload = re.sub(r"\s+", "", match.group(1))

    try:
        return base64.b64decode(payload, validate=True)
    except Exception as exc:
        raise SystemExit(
            f"ERROR: failed decoding certificate {path}: {exc}"
        )

for variable, (pem_path, hash_field) in certs.items():
    if not pem_path.is_file():
        raise SystemExit(
            f"ERROR: local Secure Boot certificate is missing: {pem_path}"
        )

    der = pem_der(pem_path)
    actual_hash = hashlib.sha256(der).hexdigest()
    recorded_hash = doc.get(hash_field)

    if actual_hash != recorded_hash:
        raise SystemExit(
            f"ERROR: local {variable} certificate does not match "
            "GjallarOS ownership metadata."
        )

    matches = glob.glob(
        f"/sys/firmware/efi/efivars/{variable}-*"
    )

    if len(matches) != 1:
        raise SystemExit(
            f"ERROR: expected exactly one active {variable} EFI variable; "
            f"found {len(matches)}."
        )

    raw = Path(matches[0]).read_bytes()

    if len(raw) < 4:
        raise SystemExit(
            f"ERROR: active {variable} EFI variable is malformed."
        )

    payload = raw[4:]

    if der not in payload:
        raise SystemExit(
            f"ERROR: active firmware {variable} does not contain "
            "the current GjallarOS certificate."
        )

print("PASS: active GjallarOS PK/KEK/db ownership verified.")
PYEOF
      '';
    };

    secureBootTool = pkgs.writeShellApplication {
      name = "gjallar-secure-boot";
      runtimeInputs = [
        pkgs.sbctl
        pkgs.systemd
        pkgs.fwupd
      ];
      text = ''
        set -euo pipefail

        case "''${1:-status}" in
          status)
            bootctl status
            printf '\n'
            sudo sbctl status
            printf '\nSigned boot artifacts:\n'
            sudo gjallar-verify-secure-boot-artifacts
            ;;
          create-keys)
            if [ -e /var/lib/sbctl/keys/db/db.key ]; then
              printf 'Secure Boot keys already exist in /var/lib/sbctl.\n' >&2
              exit 1
            fi
            printf 'Create machine-specific Secure Boot keys? Type CREATE: '
            read -r answer
            [ "$answer" = CREATE ] || exit 1
            sudo sbctl create-keys
            printf 'Keys created. Set secureBootEnable=true and rebuild before enrollment.\n'
            ;;
          enroll-framework)
            printf '%s\n' \
              'DANGER: firmware must already be in Secure Boot Setup Mode.' \
              'Framework: delete ONLY PK. Keep KEK, DB, and DBX intact.' \
              'Do not use the firmware Erase All Secure Boot Settings action.' \
              'This keeps Framework firmware-builtin keys for devices and fwupd.'
            printf 'Type ENROLL to write keys into firmware: '
            read -r answer
            [ "$answer" = ENROLL ] || exit 1
            sudo sbctl enroll-keys --firmware-builtin=db,KEK
            sudo gjallar-verify-secure-boot-artifacts
            ;;
          arm-enrollment)
            sudo gjallar-verify-secure-boot-artifacts
            sudo install -d -m 0700 /var/lib/gjallarOS
            sudo install -m 0600 /dev/null /var/lib/gjallarOS/secure-boot-enrollment-armed
            printf 'Enrollment armed. Reboot into firmware setup and enter Setup Mode.\n'
            ;;
          firmware)
            fwupdmgr get-devices
            ;;
          *)
            printf '%s\n' \
              'Usage: gjallar-secure-boot {status|create-keys|arm-enrollment|enroll-framework|firmware}'
            exit 2
            ;;
        esac
      '';
    };
  in
  {
    environment.etc."fw-fanctrl/config.json".text = configJson;
    # Secure Boot key existence is validated/provisioned by the installer.
    # Do not inspect /var/lib/sbctl with builtins.pathExists here: flake
    # evaluation must not depend on mutable host filesystem state.

    environment.systemPackages = [
      pkgs.fw-fanctrl
      pkgs.sbctl
      secureBootTool
      secureBootArtifactVerifier
      secureBootOwnershipVerifier
    ];

    # The Framework USB-C expansion-card controller at c1:00.3 can fail its
    # D0 -> D3hot transition, after which the USB 2 companion repeatedly logs
    # "Cannot enable" and the Realtek LAN card (0bda:8156) disappears. Keep
    # only this controller and that adapter out of runtime power saving.
    services.udev.extraRules = ''
      ACTION=="add|bind", SUBSYSTEM=="pci", ATTR{vendor}=="0x1022", ATTR{device}=="0x15b9", ATTR{subsystem_vendor}=="0xf111", ATTR{subsystem_device}=="0x0006", TEST=="power/control", ATTR{power/control}="on"
      ACTION=="add|bind", SUBSYSTEM=="usb", ATTR{idVendor}=="0bda", ATTR{idProduct}=="8156", TEST=="power/control", ATTR{power/control}="on"
    '';

    boot.loader.grub.enable = lib.mkIf settings.secureBootEnable (lib.mkForce false);
    boot.loader.systemd-boot.enable = lib.mkIf settings.secureBootEnable (lib.mkForce false);
    boot.lanzaboote = lib.mkIf settings.secureBootEnable {
      enable = true;
      pkiBundle = "/var/lib/sbctl";
      configurationLimit = 8;
    };
    systemd.services.gjallar-secure-boot-enroll = lib.mkIf settings.secureBootEnable {
      description = "Enroll GjallarOS Secure Boot keys in firmware Setup Mode";
      wantedBy = [ "multi-user.target" ];
      after = [ "local-fs.target" ];
      unitConfig.ConditionPathExists = "/var/lib/gjallarOS/secure-boot-enrollment-armed";

      serviceConfig = {
        Type = "oneshot";
        UMask = "0077";
      };

      path = [
        pkgs.sbctl
        pkgs.coreutils
        pkgs.gnugrep
        pkgs.systemd
        pkgs.python3
        pkgs.util-linux
          pkgs.e2fsprogs
        secureBootArtifactVerifier
        secureBootOwnershipVerifier
      ];

      script = ''
        set -euo pipefail
        export LC_ALL=C

        enrollment_marker=/var/lib/gjallarOS/secure-boot-enrollment-armed
        ownership=/var/lib/gjallarOS/secure-boot/ownership.json
        final_marker=/var/lib/gjallarOS/secure-boot-enable-required

        status="$(sbctl status)"

        if ! printf '%s\n' "$status" | grep -Eq 'Setup Mode:.*Enabled'; then
          printf '%s\n' \
            'Enrollment remains armed: firmware is not in Setup Mode.' \
            'Clear ONLY PK in firmware.' \
            'Keep KEK, db, and dbx intact; never use Erase All Secure Boot Settings.' >&2
          exit 0
        fi

        if ! ls /sys/firmware/efi/efivars/KEK-* >/dev/null 2>&1; then
          printf '%s\n' 'ERROR: KEK is missing. Restore factory Secure Boot keys and restart provisioning.' >&2
          exit 1
        fi

        if ! ls /sys/firmware/efi/efivars/db-* >/dev/null 2>&1; then
          printf '%s\n' 'ERROR: db is missing. Restore factory Secure Boot keys and restart provisioning.' >&2
          exit 1
        fi

        if ! ls /sys/firmware/efi/efivars/dbx-* >/dev/null 2>&1; then
          printf '%s\n' 'ERROR: dbx is missing. Restore factory Secure Boot keys and restart provisioning.' >&2
          exit 1
        fi

        if [ ! -s "$ownership" ]; then
          printf '%s\n' \
            'ERROR: firmware is in Setup Mode, but GjallarOS ownership metadata is missing.' \
            'Refusing automatic key enrollment.' >&2
          exit 1
        fi

        # efivarfs commonly protects existing Secure Boot variables with the
        # Linux immutable bit. sbctl refuses enrollment while that protection is
        # present, so clear it only after Setup Mode and KEK/db/dbx preservation
        # have already been verified above.
        #
        # PK is absent here by design: deleting only PK is what placed firmware
        # into Setup Mode. Never require or fabricate a PK efivar before sbctl
        # installs the GjallarOS Platform Key.
        chattr -i           /sys/firmware/efi/efivars/KEK-*           /sys/firmware/efi/efivars/db-*

        # Explicitly import firmware KEK + db only.
        # The firmware PK must never be inherited: GjallarOS becomes the
        # Platform Key owner while retaining Framework/Microsoft compatibility.
        sbctl enroll-keys --firmware-builtin=db,KEK

        gjallar-verify-secure-boot-artifacts

        status="$(sbctl status)"

        if printf '%s\n' "$status" | grep -Eq 'Setup Mode:.*Enabled'; then
          printf '%s\n' \
            'ERROR: key enrollment completed without leaving Setup Mode.' >&2
          exit 1
        fi

        # The active hierarchy now needs to contain our generated PK/KEK/db.
        # sbctl status/verify succeeded above; persist the state transition
        # without regenerating or replacing any keys.
        python3 - "$ownership" <<'PYEOF'
import json
import os
import sys
import tempfile
from datetime import datetime, timezone

path = sys.argv[1]

with open(path, "r", encoding="utf-8") as f:
    doc = json.load(f)

if doc.get("schema") != 1 or doc.get("manager") != "gjallarOS":
    raise SystemExit("invalid GjallarOS Secure Boot ownership record")

doc["stage"] = "enrolled"
doc["updatedAt"] = datetime.now(timezone.utc).isoformat().replace("+00:00", "Z")

directory = os.path.dirname(path)
fd, tmp = tempfile.mkstemp(prefix=".ownership-", dir=directory)

try:
    os.fchmod(fd, 0o600)
    with os.fdopen(fd, "w", encoding="utf-8") as f:
        json.dump(doc, f, indent=2)
        f.write("\n")
        f.flush()
        os.fsync(f.fileno())
    os.replace(tmp, path)
finally:
    if os.path.exists(tmp):
        os.unlink(tmp)
PYEOF

        gjallar-verify-secure-boot-ownership enrolled

        install -m 0600 /dev/null "$final_marker"

        # Remove this FIRST so a reboot can never re-enroll in a loop.
        rm -f -- "$enrollment_marker"

        # Recovery confirmation is only an installer-resume checkpoint.
        # It is no longer needed once firmware ownership is enrolled.
        rm -f --           /var/lib/gjallarOS/secure-boot-recovery-confirmed           /var/lib/gjallarOS/secure-boot/recovery-passphrase.pending

        printf '%s\n' \
          'GjallarOS Secure Boot ownership enrolled successfully.' \
          'GjallarOS PK is now the platform owner.' \
          'Framework/Microsoft firmware KEK/db trust was retained.' \
          'dbx was not modified.' \
          'Rebooting directly into firmware setup.' \
          'FINAL ADMIN ACTION: enable Secure Boot, save, and boot GjallarOS.'

        sync
        systemctl reboot --firmware-setup
      '';
    };

    systemd.services.gjallar-secure-boot-finalize = lib.mkIf settings.secureBootEnable {
      description = "Verify final GjallarOS Secure Boot activation";
      wantedBy = [ "multi-user.target" ];
      after = [
        "local-fs.target"
        "systemd-remount-fs.service"
      ];
      unitConfig.ConditionPathExists = "/var/lib/gjallarOS/secure-boot-enable-required";

      serviceConfig = {
        Type = "oneshot";
        UMask = "0077";
      };

      path = [
        pkgs.sbctl
        pkgs.coreutils
        pkgs.gnugrep
        secureBootArtifactVerifier
        secureBootOwnershipVerifier
      ];

      script = ''
        set -euo pipefail
        export LC_ALL=C

        marker=/var/lib/gjallarOS/secure-boot-enable-required

        status=""
for attempt in 1 2 3 4 5; do
  if status="$(sbctl status 2>&1)"; then
    break
  fi

  if [ "$attempt" -eq 5 ]; then
    printf '%s\n' \
      'ERROR: sbctl status was not ready after 5 attempts during boot.' \
      "$status" >&2
    exit 1
  fi

  sleep 1
done

        if ! printf '%s\n' "$status" | grep -Eq 'Secure Boot:.*Enabled'; then
          printf '%s\n' \
            'GjallarOS Secure Boot keys are enrolled, but Secure Boot enforcement is still disabled.' \
            'Enter firmware setup and enable Secure Boot.' >&2
          exit 0
        fi

        if printf '%s\n' "$status" | grep -Eq 'Setup Mode:.*Enabled'; then
          printf '%s\n' \
            'ERROR: Secure Boot reports enabled while firmware remains in Setup Mode.' >&2
          exit 1
        fi

        gjallar-verify-secure-boot-artifacts
        gjallar-verify-secure-boot-ownership enrolled

        rm -f -- "$marker"

        printf '%s\n' \
          'GjallarOS Secure Boot provisioning complete.' \
          'Secure Boot is enabled, Setup Mode is disabled, and signed artifacts verified.'
      '';
    };

    systemd.services.gjallar-installer-post-secure-boot = lib.mkIf settings.secureBootEnable {
      description = "Finish GjallarOS installer after Secure Boot provisioning";

      wantedBy = [ "multi-user.target" ];

      after = [
        "local-fs.target"
        "systemd-remount-fs.service"
        "gjallar-secure-boot-finalize.service"
      ];

      unitConfig.ConditionPathExists =
        "/var/lib/gjallarOS/installer-resume-after-secure-boot";

      serviceConfig = {
        Type = "oneshot";
        UMask = "0077";
      };

      path = [
        pkgs.sbctl
        pkgs.coreutils
        pkgs.gnugrep
        pkgs.gawk
        pkgs.systemd
        pkgs.util-linux
        pkgs.zenity
        secureBootArtifactVerifier
        secureBootOwnershipVerifier
      ];

      script = ''
        set -euo pipefail
        export LC_ALL=C

        resume=/var/lib/gjallarOS/installer-resume-after-secure-boot
        sb_final=/var/lib/gjallarOS/secure-boot-enable-required
        status_file=/var/lib/gjallarOS/installer-status.txt
        complete=/var/lib/gjallarOS/installation-complete

        write_status() {
          message="$1"

          install -d -m 0700 /var/lib/gjallarOS

          printf '%s\n' "$message" > "$status_file"
          chmod 0644 "$status_file"

          printf '%s\n' "$message"

          # Always make progress visible on logged-in terminals.
          printf '\n[GjallarOS Installer]\n%s\n\n' "$message" |
            wall 2>/dev/null || true
        }

        gtk_message() {
          message="$1"
          timeout_seconds="''${2:-0}"
          gtk_log=/var/lib/gjallarOS/installer-gtk.log
          launched=0

          # Do not attempt to manufacture a Wayland session from the root
          # service. The logged-in user's systemd manager already owns the
          # authoritative graphical environment (WAYLAND_DISPLAY, DISPLAY,
          # XDG_CURRENT_DESKTOP, etc.).
          #
          # Spawn Zenity as a transient USER unit. This also moves the dialog
          # out of this root oneshot service's cgroup, so systemd does not kill
          # the popup when the installer finisher exits.
          for runtime in /run/user/[0-9]*; do
            [ -d "$runtime" ] || continue
            [ -S "$runtime/bus" ] || continue

            uid="''${runtime##*/}"

            user="$(
              loginctl list-users --no-legend 2>/dev/null |
                awk -v uid="$uid" '$1 == uid { print $2; exit }'
            )"

            [ -n "$user" ] || continue

            unit="gjallar-installer-notice-$(date +%s%N)"

            if [ "$timeout_seconds" -gt 0 ]; then
              if runuser -u "$user" -- \
                env \
                  XDG_RUNTIME_DIR="$runtime" \
                  DBUS_SESSION_BUS_ADDRESS="unix:path=$runtime/bus" \
                systemd-run \
                  --user \
                  --quiet \
                  --collect \
                  --unit="$unit" \
                  ${lib.getExe pkgs.zenity} \
                    --info \
                    --timeout="$timeout_seconds" \
                    --title="GjallarOS Installer" \
                    --width=520 \
                    --text="$message"
              then
                printf '%s
' \
                  "GTK installer notification launched for $user ($unit)." \
                  >> "$gtk_log"
                launched=1
              else
                rc=$?
                printf '%s
' \
                  "GTK installer notification FAILED for $user (rc=$rc)." \
                  >> "$gtk_log"
              fi
            else
              if runuser -u "$user" -- \
                env \
                  XDG_RUNTIME_DIR="$runtime" \
                  DBUS_SESSION_BUS_ADDRESS="unix:path=$runtime/bus" \
                systemd-run \
                  --user \
                  --quiet \
                  --collect \
                  --unit="$unit" \
                  ${lib.getExe pkgs.zenity} \
                    --info \
                    --title="GjallarOS Installer" \
                    --width=520 \
                    --text="$message"
              then
                printf '%s
' \
                  "GTK installer notification launched for $user ($unit)." \
                  >> "$gtk_log"
                launched=1
              else
                rc=$?
                printf '%s
' \
                  "GTK installer notification FAILED for $user (rc=$rc)." \
                  >> "$gtk_log"
              fi
            fi

            # One physical graphical desktop is enough.
            break
          done

          if [ "$launched" -eq 0 ]; then
            printf '%s
' \
              'No usable graphical user session found; terminal/journal status remains available.' \
              >> "$gtk_log"
          fi

          # GUI notification failure must never invalidate an otherwise
          # successful installation transaction.
          return 0
        }

        fail_visible() {
          message="$1"

          write_status "INSTALLER PAUSED: $message"
          gtk_message "GjallarOS installation is paused.

$message

The continuation marker was kept so the operation can be resumed safely."

          exit 1
        }

        write_status \
          "Secure Boot is enabled. GjallarOS is running final installer verification."

        gtk_message "Secure Boot is enabled.

GjallarOS is now finishing installation and verifying the boot chain.

No action is required unless an error is shown." 5

        # The Secure Boot finalizer must have completed first.
        if [ -e "$sb_final" ]; then
          fail_visible \
            "Secure Boot final verification has not completed successfully yet."
        fi

        status="$(sbctl status 2>&1)" ||
          fail_visible "Unable to read Secure Boot status."

        printf '%s\n' "$status" |
          grep -Eq 'Secure Boot:.*Enabled' ||
          fail_visible "Secure Boot is not enforcing."

        if printf '%s\n' "$status" |
          grep -Eq 'Setup Mode:.*Enabled'; then
          fail_visible "Firmware is unexpectedly still in Secure Boot Setup Mode."
        fi

        write_status "Verifying signed GjallarOS boot artifacts..."

        gjallar-verify-secure-boot-artifacts ||
          fail_visible "GjallarOS boot artifact verification failed."

        write_status "Verifying GjallarOS Secure Boot ownership..."

        gjallar-verify-secure-boot-ownership enrolled ||
          fail_visible "GjallarOS PK/KEK/db ownership verification failed."

        # Basic installed-system sanity. Do not declare completion unless the
        # active system and booted-system links exist and are readable.
        [ -e /run/current-system ] ||
          fail_visible "/run/current-system is missing."

        [ -e /run/booted-system ] ||
          fail_visible "/run/booted-system is missing."

        write_status "Final GjallarOS installer checks passed."

        # Commit completion before deleting the continuation marker.
        install -m 0600 /dev/null "$complete"
        rm -f -- "$resume"

        sync

        write_status "GjallarOS installation complete."

        gtk_message "GjallarOS installation complete.

Secure Boot is enabled.
GjallarOS PK/KEK/db ownership is verified.
Boot artifacts are verified.

The installation transaction has finished successfully."
      '';
    };

    systemd.services.gjallar-framework-fan-curve = {
      description = "Framework fan curve control";
      wantedBy = [ "multi-user.target" ];
      serviceConfig = {
        Type = "simple";
        Restart = "always";
        RestartSec = "2s";
        ExecStart = "${pkgs.fw-fanctrl}/bin/fw-fanctrl run --config /etc/fw-fanctrl/config.json --silent ${profile.defaultStrategy}";
      };
    };
    systemd.services.gjallar-framework-battery-threshold = {
      description = "Apply Framework battery charge thresholds";
      wantedBy = [ "multi-user.target" ];
      serviceConfig.Type = "oneshot";
      script = ''
        set -eu
        for battery in /sys/class/power_supply/BAT*; do
            [ -d "$battery" ] || continue
            [ ! -w "$battery/charge_control_start_threshold" ] || printf '%s\n' ${toString profile.batteryStart} > "$battery/charge_control_start_threshold"
            [ ! -w "$battery/charge_control_end_threshold" ] || printf '%s\n' ${toString profile.batteryEnd} > "$battery/charge_control_end_threshold"
        done
      '';
    };
  }
else
  { }
