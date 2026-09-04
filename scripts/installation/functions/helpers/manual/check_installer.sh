#!/usr/bin/env bash

# Compatibility helper for callers using the historical underscored filename.
# Keep its behavior identical to check-installer.sh while callers migrate.
check_installer() {
    local repo_root="${1:-${REPO_ROOT:-$PWD}}"

    command -v gjallarctl >/dev/null 2>&1 || {
        printf '%s\n' 'ERROR: gjallarctl is unavailable; rebuild GjallarOS first.' >&2
        return 127
    }

    gjallarctl check --repo "$repo_root"
}
