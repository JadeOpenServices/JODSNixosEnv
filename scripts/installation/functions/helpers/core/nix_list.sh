nix_list() {
    local value
    printf '['
    for value in "$@"; do
        printf ' "%s"' "$(nix_string "$value")"
    done
    printf ' ]'
}
