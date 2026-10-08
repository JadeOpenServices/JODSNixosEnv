#!/usr/bin/env bash
# usage: boot-first.sh ESP_MOUNTPOINT
#
# Make the systemd-boot entry of the ESP mounted at ESP_MOUNTPOINT the first
# UEFI boot target, recreating it when missing, with GjallarOS Recovery right
# behind it and every other entry after. On the HP (2026-10-08) BootOrder
# held neither Linux Boot Manager entry, so every boot landed in recovery,
# and bootctl install appended the restored entry behind USB, PXE and
# recovery. Never deletes an entry.
set -euo pipefail

[ "$#" -eq 1 ] || { echo "usage: $0 ESP_MOUNTPOINT" >&2; exit 2; }
mountpoint=$1
systemd_boot='\EFI\systemd\systemd-bootx64.efi'

esp=$(findmnt -nro SOURCE --mountpoint "$mountpoint" || true)
[ -n "$esp" ] && [ -f "$mountpoint/EFI/systemd/systemd-bootx64.efi" ] || {
  echo "ERROR: $mountpoint is not a mounted ESP with systemd-boot" >&2
  exit 1
}
esp_partuuid=$(lsblk -dnro PARTUUID "$esp" | tr '[:upper:]' '[:lower:]')

boot_entries() {
  efibootmgr -v | tr '[:upper:]' '[:lower:]' |
    grep -F "gpt,$esp_partuuid," |
    grep -F '\efi\systemd\systemd-bootx64.efi' |
    sed -n 's/^boot\([0-9a-f]\{4\}\).*/\1/p'
  true
}

esp_entries=$(boot_entries | paste -sd, -)
if [ -z "$esp_entries" ]; then
  efibootmgr \
    --create-only \
    --disk "/dev/$(lsblk -dnro PKNAME "$esp")" \
    --part "$(cat "/sys/class/block/$(basename "$(realpath "$esp")")/partition")" \
    --label 'Linux Boot Manager' \
    --loader "$systemd_boot" >/dev/null
  esp_entries=$(boot_entries | paste -sd, -)
  [ -n "$esp_entries" ] || {
    echo "ERROR: systemd-boot UEFI boot entry for $esp was not created" >&2
    exit 1
  }
fi

recovery_entries=$(efibootmgr | tr '[:upper:]' '[:lower:]' |
  sed -n 's/^boot\([0-9a-f]\{4\}\)[* ] gjallaros recovery\(\t.*\)\{0,1\}$/\1/p' | paste -sd, -)
rest=$(efibootmgr | sed -n 's/^BootOrder: //p' | tr '[:upper:]' '[:lower:]' | tr , '\n' |
  grep -vxF -f <(printf '%s\n' "$esp_entries,$recovery_entries" | tr , '\n' | grep .) |
  paste -sd, - || true)
new_order=$(printf '%s\n' "$esp_entries" "$recovery_entries" "$rest" | grep . | paste -sd, -)
efibootmgr --bootorder "$new_order" >/dev/null

first=$(efibootmgr | sed -n 's/^BootOrder: //p' | cut -d, -f1 | tr '[:upper:]' '[:lower:]')
case ",$esp_entries," in
  *",$first,"*) ;;
  *)
    echo "ERROR: firmware did not keep systemd-boot first in BootOrder" >&2
    exit 1
    ;;
esac
echo "PASS: BootOrder $(efibootmgr | sed -n 's/^BootOrder: //p')"
