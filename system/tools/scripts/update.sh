#!/usr/bin/env bash
set -euo pipefail
repo="${GJALLAROS_REPO:-__REPO_ROOT__}"
host="${GJALLAROS_HOST:-__HOSTNAME__}"
cd "$repo"
case "${1:-}" in
    --rebuild|-r) nix flake update; exec rebuild ;;
    --check) exec nix flake check ;;
    "") exec nix flake update ;;
    *) printf 'Usage: update [--rebuild|-r|--check]\n' >&2; exit 2 ;;
esac
