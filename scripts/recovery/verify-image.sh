#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "usage: $0 IMAGE MANIFEST SIGNATURE PINNED_ED25519_PUBLIC_KEY" >&2
  exit 2
}

[ "$#" -eq 4 ] || usage
image=$(realpath "$1")
manifest=$(realpath "$2")
signature=$(realpath "$3")
public_key=$(realpath "$4")

openssl pkeyutl -verify -rawin -pubin -inkey "$public_key" \
  -in "$manifest" -sigfile "$signature" >/dev/null

expected_name=$(sed -n 's/^image=//p' "$manifest")
expected_sha=$(sed -n 's/^sha256=//p' "$manifest")
expected_size=$(sed -n 's/^size=//p' "$manifest")
[ "$(basename "$image")" = "$expected_name" ] || { echo "ERROR: image name mismatch" >&2; exit 1; }
[ "$(stat -c '%s' "$image")" = "$expected_size" ] || { echo "ERROR: image size mismatch" >&2; exit 1; }
printf '%s  %s\n' "$expected_sha" "$image" | sha256sum --check --status || {
  echo "ERROR: image digest mismatch" >&2
  exit 1
}
echo "Recovery image signature and digest verified."
