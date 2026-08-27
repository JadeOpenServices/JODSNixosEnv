nix_string() {
    printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}
