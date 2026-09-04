#!/usr/bin/env bash
# Small bootstrap only. The installer itself is cmd/gjallar-installer.
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo="$(cd "$script_dir/../.." && pwd)"

command -v nix >/dev/null 2>&1 || {
    printf 'ERROR: Nix is required to bootstrap GjallarOS.\n' >&2
    exit 1
}

exec nix --extra-experimental-features 'nix-command flakes' shell \
    "path:$repo#gjallar-installer" \
    nixpkgs#go_1_26 nixpkgs#pciutils nixpkgs#git nixpkgs#fwupd \
    nixpkgs#curl nixpkgs#zenity nixpkgs#sops nixpkgs#age \
    nixpkgs#whois nixpkgs#openssl nixpkgs#cryptsetup \
    nixpkgs#util-linux nixpkgs#systemd \
    -c gjallar-installer --repo "$repo" "$@"
