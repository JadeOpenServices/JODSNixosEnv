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

# Nix evaluates a staged copy of the checkout: a path: reference to the
# checkout itself copies .git (gigabytes of history) and ignored scratch such
# as .vm into the store, which is RAM on live media. Same exclusions as
# internal/installer/flakesource.
flake_tmp="$(mktemp -d "${TMPDIR:-/tmp}/gjallar-flake-XXXXXX")"
trap 'rm -rf "$flake_tmp"' EXIT
flake_src="$flake_tmp/source"
stage_flake() {
    rm -rf "$flake_src"
    mkdir "$flake_src"
    tar -C "$repo" --exclude=.git --anchored --exclude=./.vm --exclude=./.claude \
        --exclude=./.direnv --exclude=./result --exclude='./result-*' -cf - . |
        tar -C "$flake_src" -xf -
}
stage_flake

# A plain git clone leaves submodules empty. Resolve required source before
# building anything or applying prerequisite changes to the installed system.
if [[ ! -f "$repo/pkgs/monique/nix/nixos-module.nix" ]]; then
    printf 'Preparing the required Monique submodule...\n'
    git_cmd=(git)
    if ! command -v git >/dev/null 2>&1; then
        git_cmd=(nix --extra-experimental-features 'nix-command flakes' shell
            --inputs-from "path:$flake_src" nixpkgs#git --command git)
    fi
    if ! "${git_cmd[@]}" -C "$repo" submodule update --init --recursive -- pkgs/monique ||
        [[ ! -f "$repo/pkgs/monique/nix/nixos-module.nix" ]]; then
        printf 'ERROR: Monique source is missing. Use git clone --recurse-submodules or copy the complete repository including pkgs/monique.\n' >&2
        exit 1
    fi
    stage_flake
fi

printf 'Building the GjallarOS installer...\n'
installer_path="$(nix --extra-experimental-features 'nix-command flakes' build \
    --no-link --print-out-paths "path:$flake_src#gjallar-installer")"

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
    # Provided by the prerequisite rebuild on installed hosts; live media
    # skips that rebuild and the minimal ISO ships none of them.
    nixpkgs#sbctl
    nixpkgs#openssl
    nixpkgs#age
    nixpkgs#sops
)

printf 'Preparing GjallarOS hardware probe environment...\n'

# Not exec: the EXIT trap removes the staged source afterwards.
nix --extra-experimental-features 'nix-command flakes' shell \
    --inputs-from "path:$flake_src" \
    "${probe_tools[@]}" \
    --command "$installer_path/bin/gjallar-installer" --repo "$repo" "$@"
