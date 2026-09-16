#!/usr/bin/env bash

# Enable the modern Nix CLI and flakes for the entire installer bootstrap.
if [ -n "${NIX_CONFIG:-}" ]; then
    NIX_CONFIG="${NIX_CONFIG}
extra-experimental-features = nix-command flakes"
    export NIX_CONFIG
else
    export NIX_CONFIG='extra-experimental-features = nix-command flakes'
fi

# Build and launch the installer without changing the caller's environment.
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo="$(cd "$script_dir/../.." && pwd)"

command -v nix >/dev/null 2>&1 || {
    printf 'ERROR: Nix is required to bootstrap GjallarOS.\n' >&2
    exit 1
}

printf 'Building the GjallarOS installer...\n'
installer_path="$(nix --extra-experimental-features 'nix-command flakes' build \
    --no-link --print-out-paths "path:$repo#gjallar-installer")"

probe_tools=(
    nixpkgs#pciutils
    nixpkgs#usbutils
    nixpkgs#util-linux
    nixpkgs#libinput
    nixpkgs#iio-sensor-proxy
    nixpkgs#fprintd
    nixpkgs#bolt
    nixpkgs#alsa-utils
    nixpkgs#pipewire
    nixpkgs#fwupd
    nixpkgs#ethtool
    nixpkgs#iw
)

printf 'Preparing GjallarOS hardware probe environment...\n'

exec nix --extra-experimental-features 'nix-command flakes' shell \
    --inputs-from "path:$repo" \
    "${probe_tools[@]}" \
    --command "$installer_path/bin/gjallar-installer" --repo "$repo" "$@"
