configure_luks() {
    local hardware_file="$1" device passphrase change saved new_key recovery_key old_file new_file recovery_file
    [[ -f "$hardware_file" ]] || return 0
    grep -qE 'boot\.initrd\.luks\.devices|luks\.devices' "$hardware_file" || return 0

    say '[NOTE] LUKS mapping detected in the generated hardware configuration.'
    command -v cryptsetup >/dev/null 2>&1 || {
        say '[WARN] cryptsetup is unavailable; skipping LUKS key checks.'
        return 0
    }
    command -v sudo >/dev/null 2>&1 || { say '[WARN] sudo is unavailable; skipping LUKS key checks.'; return 0; }
    device="$(lsblk -pnro NAME,FSTYPE 2>/dev/null | awk '$2 == "crypto_LUKS" {print $1; exit}')"
    [[ -n "$device" ]] || {
        say '[WARN] No active crypto_LUKS device was found; skipping key changes.'
        return 0
    }
    read -r -s -p "Enter the current LUKS key for $device (empty to skip): " passphrase; printf '\n'
    [[ -n "$passphrase" ]] || return 0
    if ! printf '%s' "$passphrase" | sudo cryptsetup open --test-passphrase --type luks "$device" --key-file - 2>/dev/null; then
        unset passphrase
        die 'The supplied LUKS key was rejected; no disk changes were made.'
    fi
    read -r -p 'Change the LUKS key? [y/N] ' change
    new_key=""
    [[ "$change" =~ ^([yY]|[yY][eE][sS])$ ]] && new_key="$(openssl rand -hex 32 2>/dev/null || od -An -N32 -tx1 /dev/urandom | tr -d ' \n')"
    recovery_key="$(openssl rand -hex 32 2>/dev/null || od -An -N32 -tx1 /dev/urandom | tr -d ' \n')"
    [[ -n "$new_key" ]] && printf '\nNew LUKS key (64 characters; save this):\n%s\n' "$new_key"
    printf '\nRecovery key (64 characters; save this separately):\n%s\n' "$recovery_key"
    read -r -p 'Saved the displayed key(s) securely? [y/N] ' saved
    if [[ ! "$saved" =~ ^([yY]|[yY][eE][sS])$ ]]; then
        read -r -p 'The key(s) cannot be recovered after they are lost. Saved them now? [y/N] ' saved
        [[ "$saved" =~ ^([yY]|[yY][eE][sS])$ ]] || { unset passphrase new_key recovery_key; say 'Keeping the existing LUKS key.'; return 0; }
    fi
    old_file="$(mktemp)"; new_file="$(mktemp)"; recovery_file="$(mktemp)"
    chmod 600 "$old_file" "$new_file" "$recovery_file"
    printf '%s' "$passphrase" > "$old_file"
    printf '%s' "$recovery_key" > "$recovery_file"
    if sudo cryptsetup luksAddKey "$device" "$recovery_file" --key-file "$old_file"; then
        say 'Recovery key enrolled successfully.'
        if [[ -n "$new_key" ]]; then
            printf '%s' "$new_key" > "$new_file"
            if sudo cryptsetup luksChangeKey "$device" "$new_file" --key-file "$old_file"; then
                say 'LUKS key changed successfully. Keep both displayed keys safe.'
            else
                say '[ERROR] LUKS key change failed; existing and recovery keys remain valid.'
            fi
        else
            say 'Existing LUKS key kept unchanged.'
        fi
    else
        say '[ERROR] Recovery key enrollment failed; no LUKS key was changed.'
    fi
    rm -f -- "$old_file" "$new_file" "$recovery_file"
    unset passphrase new_key recovery_key
}
