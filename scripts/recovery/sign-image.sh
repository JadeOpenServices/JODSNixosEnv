#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "usage: $0 IMAGE ED25519_PRIVATE_KEY OUTPUT_DIR" >&2
  exit 2
}

[ "$#" -eq 3 ] || usage
image=$(realpath "$1")
private_key=$(realpath "$2")
output_dir=$3
[ -f "$image" ] || { echo "ERROR: image not found: $image" >&2; exit 1; }
[ -f "$private_key" ] || { echo "ERROR: signing key not found: $private_key" >&2; exit 1; }
[ "$(stat -c '%a' "$private_key")" = 600 ] || {
  echo "ERROR: signing key must have mode 0600" >&2
  exit 1
}

mkdir -p "$output_dir"
manifest="$output_dir/$(basename "$image").manifest"
signature="$manifest.sig"
public_key="$output_dir/recovery-signing-public.pem"

sha256=$(sha256sum "$image" | cut -d' ' -f1)
size=$(stat -c '%s' "$image")
revision=${GJALLAROS_REVISION:-$(git -C "$(dirname "$0")/../.." rev-parse HEAD)}
created=$(date -u +%Y-%m-%dT%H:%M:%SZ)

umask 077
printf 'schema=1\nimage=%s\nsha256=%s\nsize=%s\nrevision=%s\ncreated=%s\n' \
  "$(basename "$image")" "$sha256" "$size" "$revision" "$created" >"$manifest"
openssl pkeyutl -sign -rawin -inkey "$private_key" -in "$manifest" -out "$signature"
openssl pkey -in "$private_key" -pubout -out "$public_key"
chmod 0644 "$manifest" "$signature" "$public_key"
echo "Signed recovery release: $manifest"
