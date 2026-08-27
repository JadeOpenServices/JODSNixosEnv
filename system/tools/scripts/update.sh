#!/usr/bin/env bash
set -euo pipefail
repo="${ALFHEIMOS_REPO:-__REPO_ROOT__}"
host="${ALFHEIMOS_HOST:-__HOSTNAME__}"
cd "$repo"
case "${1:-}" in
    --rebuild|-r) nix flake update; exec sudo nixos-rebuild switch --flake "$repo#$host" ;;
    --check) exec nix flake check ;;
    "") exec nix flake update ;;
    *) printf 'Usage: update [--rebuild|-r|--check]\n' >&2; exit 2 ;;
esac
