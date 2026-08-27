die() {
    printf 'Error: %s\n' "$*" >&2
    exit 1
}
