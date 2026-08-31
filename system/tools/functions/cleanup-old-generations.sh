cleanup_old_generations() {
    local generations count keep=5

    generations="$(
        sudo nix-env -p /nix/var/nix/profiles/system --list-generations |
            awk '{print $1}' |
            sort -n
    )"

    count="$(printf '%s\n' "$generations" | grep -c '^[0-9]' || true)"

    if (( count <= keep )); then
        printf '[GjallarOS] Nothing to clean (%d generations, keeping %d).\n' "$count" "$keep"
        return 0
    fi

    printf '[GjallarOS] Removing %d old generations; keeping %d.\n' "$((count - keep))" "$keep"

    printf '%s\n' "$generations" |
        head -n "$((count - keep))" |
        xargs -r sudo nix-env -p /nix/var/nix/profiles/system --delete-generations
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    cleanup_old_generations "$@"
fi
