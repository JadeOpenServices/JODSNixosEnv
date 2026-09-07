#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf 'GJALLAR_RECOVERY_ERROR=%s\n' "$*" >&2
  return 1
}

[ "$(id -u)" -eq 0 ] || fail 'executor requires root'

contract="${1:-}"
artifact="${2:-}"

[ -n "$contract" ] || fail 'contract path required'
[ -n "$artifact" ] || fail 'artifact path required'
[ -s "$contract" ] || fail 'contract does not exist or is empty'
[ -s "$artifact" ] || fail 'artifact does not exist or is empty'

# This binary is intentionally recovery-only.
[ -e /etc/gjallar/recovery ] || [ -e /etc/gjallar/recovery-image ] || \
  fail 'not running inside a GjallarOS recovery environment'

for command in jq sha256sum lsblk blkid sgdisk realpath findmnt; do
  command -v "$command" >/dev/null || fail "required command missing: $command"
done

jq -e '
  .version == 1 and
  (.command.payload.schema_version == 1) and
  (
    .command.payload.operation == "reinstall" or
    .command.payload.operation == "factory-wipe"
  ) and
  (.command.payload.job_id | type == "string" and length > 0) and
  (.command.payload.registration_id | type == "string" and length > 0) and
  (.command.payload.nonce | test("^[0-9a-fA-F]{64}$")) and
  (.command.payload.artifact_digests.os_image.sha256 |
    test("^[0-9a-fA-F]{64}$")) and
  (.command.payload.artifact_digests.target_disk.gpt_disk_guid |
    test("^[0-9a-fA-F-]{36}$")) and
  (.command.payload.artifact_digests.target_disk.recovery_partition_uuid |
    test("^[0-9a-fA-F-]{36}$"))
' "$contract" >/dev/null || fail 'contract schema is invalid'

operation="$(jq -r '.command.payload.operation' "$contract")"
job_id="$(jq -r '.command.payload.job_id' "$contract")"

expected_sha="$(jq -r \
  '.command.payload.artifact_digests.os_image.sha256' \
  "$contract" | tr '[:upper:]' '[:lower:]')"

actual_sha="$(sha256sum "$artifact" | cut -d ' ' -f 1)"
[ "$actual_sha" = "$expected_sha" ] || fail 'OS artifact digest mismatch'

expected_disk_guid="$(jq -r \
  '.command.payload.artifact_digests.target_disk.gpt_disk_guid' \
  "$contract" | tr '[:upper:]' '[:lower:]')"

expected_recovery_uuid="$(jq -r \
  '.command.payload.artifact_digests.target_disk.recovery_partition_uuid' \
  "$contract" | tr '[:upper:]' '[:lower:]')"

expected_wwn="$(jq -r \
  '.command.payload.artifact_digests.target_disk.wwn // empty' \
  "$contract")"

expected_serial="$(jq -r \
  '.command.payload.artifact_digests.target_disk.serial // empty' \
  "$contract")"

partition_type='bc13c2ff-59e6-4262-a352-b275fd6f7172'

mapfile -t recovery_partitions < <(
  lsblk -pnro PATH,PARTLABEL,PARTTYPE |
    awk -v type="$partition_type" \
      '$2 == "JODS-RECOVERY" && tolower($3) == type { print $1 }'
)

[ "${#recovery_partitions[@]}" -eq 1 ] || \
  fail 'expected exactly one JODS-RECOVERY partition'

recovery_partition="${recovery_partitions[0]}"

actual_recovery_uuid="$(
  blkid -s PARTUUID -o value "$recovery_partition" |
    tr '[:upper:]' '[:lower:]'
)"

[ "$actual_recovery_uuid" = "$expected_recovery_uuid" ] || \
  fail 'recovery partition UUID mismatch'

parent_name="$(lsblk -dnro PKNAME "$recovery_partition")"
[ -n "$parent_name" ] || fail 'cannot resolve recovery parent disk'

target_disk="$(realpath "/dev/$parent_name")"
[ "$(lsblk -dnro TYPE "$target_disk")" = disk ] || \
  fail 'target is not a whole disk'

actual_disk_guid="$(
  sgdisk -p "$target_disk" |
    awk -F: '/Disk identifier \(GUID\)/ {
      gsub(/[[:space:]]/, "", $2)
      print tolower($2)
    }'
)"

[ "$actual_disk_guid" = "$expected_disk_guid" ] || \
  fail 'GPT disk GUID mismatch'

actual_wwn="$(lsblk -dnro WWN "$target_disk" | xargs)"
actual_serial="$(lsblk -dnro SERIAL "$target_disk" | xargs)"

if [ -n "$expected_wwn" ]; then
  [ "$actual_wwn" = "$expected_wwn" ] || fail 'disk WWN mismatch'
else
  [ -n "$expected_serial" ] || fail 'signed contract contains no disk identity'
  [ "$actual_serial" = "$expected_serial" ] || fail 'disk serial mismatch'
fi

# Do not allow an artifact to change which physical disk the signed
# authorization refers to.
printf '%s\n' \
  "GJALLAR_RECOVERY_OPERATION=$operation" \
  "GJALLAR_RECOVERY_JOB_ID=$job_id" \
  "GJALLAR_RECOVERY_TARGET_DISK=$target_disk" \
  "GJALLAR_RECOVERY_PARTITION=$recovery_partition" \
  "GJALLAR_RECOVERY_ARTIFACT_SHA256=$actual_sha" \
  'GJALLAR_RECOVERY_VALIDATION=passed'

# IMPORTANT:
# The repository currently has no canonical target-disk install primitive.
# Never replace this fail-closed boundary with inline sgdisk/mkfs/nixos-install.
#
# The next implementation must live in gjallarOS and accept ONLY the already
# validated target disk/artifact/operation from this function.
platform_executor='/run/current-system/sw/bin/gjallar-recovery-platform-install'

[ -x "$platform_executor" ] || \
  fail 'validated contract, but canonical GjallarOS platform installer is not implemented'

"$platform_executor" \
  --operation "$operation" \
  --target-disk "$target_disk" \
  --recovery-partition "$recovery_partition" \
  --artifact "$artifact" \
  --artifact-sha256 "$actual_sha" \
  --job-id "$job_id"
