configure_work_user_password() {
    local password_file password confirmation password_hash

    cfg_work_user_password_file=""
    [[ "${cfg_work_user_enable:-false}" == true ]] || return 0
    [[ -t 0 && -t 1 ]] || die 'The work-account password must be set from a terminal.'

    password_file="/var/lib/gjallarOS/passwords/${cfg_work_username}.hash"
    require_command mkpasswd
    require_command sudo

    if sudo test -s "$password_file"; then
        cfg_work_user_password_file="$password_file"
        say "Reusing the stored password hash for $cfg_work_username."
        return 0
    fi

    say "Set a password for $cfg_work_username (terminal only)."
    while true; do
        read -r -s -p "Password for $cfg_work_username: " password; printf '\n'
        read -r -s -p 'Confirm password: ' confirmation; printf '\n'
        [[ -n "$password" ]] || { say 'Password must not be empty.'; continue; }
        [[ "$password" == "$confirmation" ]] || { say 'Passwords do not match.'; continue; }

        password_hash="$(printf '%s' "$password" | mkpasswd --method=yescrypt --stdin)"
        unset password confirmation
        sudo install -d -m 0700 /var/lib/gjallarOS/passwords
        printf '%s\n' "$password_hash" | \
            sudo sh -c 'umask 077; cat > "$1"' sh "$password_file"
        unset password_hash
        cfg_work_user_password_file="$password_file"
        say "Stored password hash for $cfg_work_username."
        return 0
    done
}
