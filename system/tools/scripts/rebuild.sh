#!/usr/bin/env bash
set -euo pipefail
repo="${ALFHEIMOS_REPO:-__REPO_ROOT__}"
host="${ALFHEIMOS_HOST:-__HOSTNAME__}"
exec sudo nixos-rebuild switch --flake "$repo#$host" "$@"
