#!/usr/bin/env bash
set -euo pipefail
keep=5
while [[ $# -gt 0 ]]; do
    case "$1" in
        -k|--keep) keep="${2:?--keep requires a number}"; shift 2 ;;
        -h|--help) printf 'Usage: cleanup [--keep N]\n'; exit 0 ;;
        *) printf 'Unknown option: %s\n' "$1" >&2; exit 2 ;;
    esac
done
[[ "$keep" =~ ^[0-9]+$ ]] || { printf 'Keep count must be numeric.\n' >&2; exit 2; }
sudo nix-env --profile /nix/var/nix/profiles/system --delete-generations +"$keep"
nix store gc
sudo nix store gc
