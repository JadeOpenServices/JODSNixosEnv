die() {
    if [[ -t 2 ]]; then
        printf '\033[1;31m[ERROR]\033[0m %s\n' "$*" >&2
    else
        printf '[ERROR] %s\n' "$*" >&2
    fi
    exit 1
}
