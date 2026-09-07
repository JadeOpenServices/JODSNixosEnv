#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "usage: $0 PARTITION IMAGE MANIFEST SIGNATURE PINNED_PUBLIC_KEY" >&2
  echo "PARTITION must be a dedicated, unmounted 2-4 GiB partition." >&2
  echo "This command never accepts, creates, shrinks, or formats a whole disk." >&2
  exit 2
}

[ "$#" -eq 5 ] || usage
[ "$(id -u)" -eq 0 ] || { echo "ERROR: run as root" >&2; exit 1; }
partition=$(realpath "$1")
image=$(realpath "$2")
manifest=$(realpath "$3")
signature=$(realpath "$4")
public_key=$(realpath "$5")

[ "$(lsblk -dnro TYPE "$partition")" = part ] || {
  echo "ERROR: target must be a partition, never a whole disk" >&2
  exit 1
}
part_label=$(lsblk -dnro PARTLABEL "$partition")
[ "$part_label" = JODS-RECOVERY ] || {
  echo "ERROR: target partition must have GPT label JODS-RECOVERY" >&2
  exit 1
}
part_type=$(lsblk -dnro PARTTYPE "$partition" | tr '[:upper:]' '[:lower:]')
[ "$part_type" = bc13c2ff-59e6-4262-a352-b275fd6f7172 ] || {
  echo "ERROR: JODS-RECOVERY must use the XBOOTLDR partition type" >&2
  exit 1
}
[ -z "$(lsblk -nro MOUNTPOINTS "$partition" | tr -d '[:space:]')" ] || {
  echo "ERROR: target partition is mounted" >&2
  exit 1
}
size=$(blockdev --getsize64 "$partition")
[ "$size" -ge 2147483648 ] && [ "$size" -le 4294967296 ] || {
  echo "ERROR: recovery partition must be between 2 and 4 GiB" >&2
  exit 1
}

"$(dirname "$0")/verify-image.sh" "$image" "$manifest" "$signature" "$public_key"
for command in mkfs.vfat xorriso sbctl findmnt; do
  command -v "$command" >/dev/null || {
    echo "ERROR: required command not found: $command" >&2
    exit 1
  }
done
[ -r /var/lib/sbctl/keys/db/db.key ] && [ -r /var/lib/sbctl/keys/db/db.pem ] || {
  echo "ERROR: per-device Secure Boot signing keys are not installed" >&2
  exit 1
}
parent_name=$(lsblk -dnro PKNAME "$partition")
part_number=$(lsblk -dnro PARTN "$partition")
[ -n "$parent_name" ] && [ -n "$part_number" ] || {
  echo "ERROR: cannot resolve parent disk and partition number" >&2
  exit 1
}
parent="/dev/$parent_name"
[ "$(lsblk -dnro PTTYPE "$parent")" = gpt ] || {
  echo "ERROR: recovery partition requires a GPT disk" >&2
  exit 1
}
echo "Target: $partition ($(numfmt --to=iec "$size"))"
printf 'Type FORMAT-JODS-RECOVERY to erase that partition: '
read -r answer
[ "$answer" = FORMAT-JODS-RECOVERY ] || { echo "Cancelled." >&2; exit 1; }

# Preserve the XBOOTLDR GUID validated above. Changing it to an ESP here would
# make discovery inconsistent and could cause firmware/bootloader ambiguity.
mkfs.vfat -F 32 -n JODSRECOV "$partition"
mount_dir=$(mktemp -d)
cleanup() {
  mountpoint -q "$mount_dir" && umount "$mount_dir"
  rmdir "$mount_dir"
}
trap cleanup EXIT
mount -o nodev,nosuid,noexec "$partition" "$mount_dir"
xorriso -osirrox on -indev "$image" -extract / "$mount_dir" >/dev/null 2>&1

[ -f "$mount_dir/EFI/BOOT/BOOTX64.EFI" ] || {
  echo "ERROR: recovery image has no UEFI fallback bootloader" >&2
  exit 1
}

# Sign every executable in the copied boot chain with this endpoint's key.
# systemd-boot then verifies signed UKIs before starting them.
while IFS= read -r -d '' executable; do
  sbctl sign --save "$executable" >/dev/null
  sbctl verify "$executable" >/dev/null
done < <(find "$mount_dir/EFI" -type f -iname '*.efi' -print0)

install -d -m 0700 "$mount_dir/.gjallar-release"
install -m 0600 "$manifest" "$mount_dir/.gjallar-release/manifest"
install -m 0600 "$signature" "$mount_dir/.gjallar-release/manifest.sig"
install -m 0644 "$public_key" "$mount_dir/.gjallar-release/recovery-signing-public.pem"
sync -f "$mount_dir/EFI/BOOT/BOOTX64.EFI"
echo "Installed verified, endpoint-signed, independently bootable recovery environment."
