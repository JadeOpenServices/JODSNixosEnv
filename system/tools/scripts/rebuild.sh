#!/usr/bin/env bash
set -euo pipefail
repo="${GJALLAROS_REPO:-__REPO_ROOT__}"
host="${GJALLAROS_HOST:-__HOSTNAME__}"
messages=(
  'Checking the GjallarOS configuration.'
  'Building the next generation.'
  'Turning declarations into a working system.'
)
printf '[GjallarOS] %s\n' "${messages[RANDOM % ${#messages[@]}]}"

if command -v nom >/dev/null 2>&1; then
  sudo nixos-rebuild switch --flake "$repo#$host" --log-format internal-json "$@" | nom
else
  sudo nixos-rebuild switch --flake "$repo#$host" "$@"
fi
printf '[GjallarOS] Done. New generation is active; SDDM changes apply on next login.\n'
