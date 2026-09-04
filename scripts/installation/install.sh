#!/usr/bin/env bash
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

exec "$installer_path/bin/gjallar-installer" --repo "$repo" "$@"
