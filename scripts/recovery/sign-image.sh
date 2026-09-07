#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "usage: $0 IMAGE ED25519_PRIVATE_KEY PINNED_ED25519_PUBLIC_KEY OUTPUT_DIR" >&2
  return 2
}

if [ "$#" -ne 4 ]; then
  usage
  false
fi

image=$(realpath "$1")
private_key=$(realpath "$2")
pinned_public_key=$(realpath "$3")
output_dir=$4

[ -f "$image" ] || { echo "ERROR: image not found: $image" >&2; false; }
[ -f "$private_key" ] || { echo "ERROR: signing key not found: $private_key" >&2; false; }
[ -f "$pinned_public_key" ] || { echo "ERROR: pinned public key not found: $pinned_public_key" >&2; false; }

[ "$(stat -c '%a' "$private_key")" = 600 ] || {
  echo "ERROR: signing key must have mode 0600" >&2
  false
}

mkdir -p "$output_dir"
manifest="$output_dir/$(basename "$image").manifest"
signature="$manifest.sig"
public_key="$output_dir/recovery-signing-public.pem"

tmp_dir=$(mktemp -d)
cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT

openssl pkey -in "$private_key" -pubout -outform DER \
  > "$tmp_dir/derived.der"

openssl pkey -pubin -in "$pinned_public_key" -pubout -outform DER \
  > "$tmp_dir/pinned.der"

cmp -s "$tmp_dir/derived.der" "$tmp_dir/pinned.der" || {
  echo "ERROR: recovery private key does not match the separately pinned public key" >&2
  false
}

sha256=$(sha256sum "$image" | cut -d' ' -f1)
size=$(stat -c '%s' "$image")
revision=${GJALLAROS_REVISION:-$(git -C "$(dirname "$0")/../.." rev-parse HEAD)}
created=$(date -u +%Y-%m-%dT%H:%M:%SZ)

umask 077
printf 'schema=1\nimage=%s\nsha256=%s\nsize=%s\nrevision=%s\ncreated=%s\n' \
  "$(basename "$image")" "$sha256" "$size" "$revision" "$created" > "$manifest"

openssl pkeyutl \
  -sign \
  -rawin \
  -inkey "$private_key" \
  -in "$manifest" \
  -out "$signature"

# Persist the independently supplied trust anchor, never a newly derived key.
install -m 0644 "$pinned_public_key" "$public_key"

openssl pkeyutl \
  -verify \
  -rawin \
  -pubin \
  -inkey "$public_key" \
  -in "$manifest" \
  -sigfile "$signature" \
  >/dev/null

chmod 0644 "$manifest" "$signature" "$public_key"
echo "Signed recovery release against pinned trust anchor: $manifest"
