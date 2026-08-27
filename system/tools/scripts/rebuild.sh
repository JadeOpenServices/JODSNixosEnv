#!/usr/bin/env bash
set -euo pipefail
repo="${GJALLAROS_REPO:-__REPO_ROOT__}"
host="${GJALLAROS_HOST:-__HOSTNAME__}"
exec sudo nixos-rebuild switch --flake "$repo#$host" "$@"
