require_command() {
    command -v "$1" >/dev/null 2>&1 || die "Required command not found: $1"
}
