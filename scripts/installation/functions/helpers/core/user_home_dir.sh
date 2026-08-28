user_home_dir() {
    local username="$1" entry home
    entry="$(getent passwd "$username" 2>/dev/null || true)"
    home="$(printf '%s\n' "$entry" | awk -F: 'NR == 1 { print $6 }')"
    printf '%s\n' "${home:-/home/$username}"
}
