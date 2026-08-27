configure_luks() {
    local hardware_file="$1"
    local device=""
    local passphrase=""
    local change=""
    local saved=""
    local new_key=""
    local recovery_key=""
    local old_file=""
    local new_file=""
    local recovery_file=""

    [[ -f "$hardware_file" ]] || return 0

    grep -qE 'boot\.initrd\.luks\.devices|luks\.devices' "$hardware_file" || {
        say '[NOTE] No LUKS mapping exists in the hardware configuration; skipping LUKS configuration.'
        return 0
    }

    command -v cryptsetup >/dev/null 2>&1 || {
        say '[WARN] cryptsetup is unavailable; skipping LUKS configuration.'
        return 0
    }

    command -v sudo >/dev/null 2>&1 || {
        say '[WARN] sudo is unavailable; skipping LUKS configuration.'
        return 0
    }

    # Detect an actual LUKS device on the machine.
    device="$(
        lsblk -pnro NAME,FSTYPE 2>/dev/null |
            awk '$2 == "crypto_LUKS" {print $1; exit}'
    )"

    if [[ -z "$device" ]]; then
        say '[NOTE] No LUKS-encrypted device detected; skipping LUKS configuration.'
        return 0
    fi

    printf '\n'
    printf '%s\n' '============================================================'
    printf '%s\n' 'LUKS CONFIGURATION'
    printf '%s\n' '============================================================'
    printf 'Detected LUKS device: %s\n' "$device"
    printf '\n'

    read -r -p 'Configure LUKS keys for this device? [y/N] ' change

    if [[ ! "$change" =~ ^([yY]|[yY][eE][sS])$ ]]; then
        say 'LUKS configuration skipped; existing unlock methods remain unchanged.'
        return 0
    fi

    read -r -s \
        -p "Enter the current LUKS key for $device: " \
        passphrase
    printf '\n'

    if [[ -z "$passphrase" ]]; then
        say 'No LUKS key supplied; keeping the existing configuration.'
        unset passphrase
        return 0
    fi

    # Verify the supplied key before making ANY changes.
    if ! printf '%s' "$passphrase" |
        sudo cryptsetup open \
            --test-passphrase \
            --type luks \
            "$device" \
            --key-file - \
            2>/dev/null; then

        unset passphrase
        die 'The supplied LUKS key was rejected; no disk changes were made.'
    fi

    read -r -p 'Generate a new LUKS key? [y/N] ' change

    if [[ "$change" =~ ^([yY]|[yY][eE][sS])$ ]]; then

        new_key="$(
            openssl rand -hex 32 2>/dev/null ||
            od -An -N32 -tx1 /dev/urandom | tr -d ' \n'
        )"

        while true; do
            printf '\n'
            printf '%s\n' '============================================================'
            printf '%s\n' 'NEW LUKS KEY'
            printf '%s\n' '============================================================'
            printf '%s\n' "$new_key"
            printf '%s\n' '============================================================'
            printf '\n'
            printf '%s\n' 'IMPORTANT: Save this key somewhere secure.'
            printf '%s\n' 'It is not written to settings.nix or the repository.'
            printf '%s\n' 'The installer cannot recover it after this session.'
            printf '\n'

            read -r -p 'Have you saved the new LUKS key? [y/N/q] ' saved

            if [[ "$saved" =~ ^([qQ])$ ]]; then
                unset passphrase new_key recovery_key
                say 'LUKS rotation cancelled; existing key remains unchanged.'
                return 0
            fi

            if [[ "$saved" =~ ^([yY]|[yY][eE][sS])$ ]]; then
                read -r -p 'Are you absolutely sure you saved the new LUKS key? [y/N/q] ' saved

                if [[ "$saved" =~ ^([qQ])$ ]]; then
                    unset passphrase new_key recovery_key
                    say 'LUKS rotation cancelled; existing key remains unchanged.'
                    return 0
                fi

                if [[ "$saved" =~ ^([yY]|[yY][eE][sS])$ ]]; then
                    break
                fi
            fi

            printf '\n'
            say 'You must confirm that the new LUKS key has been saved.'
            say 'The key will be displayed again.'
        done

        recovery_key="$(
            openssl rand -hex 32 2>/dev/null ||
            od -An -N32 -tx1 /dev/urandom | tr -d ' \n'
        )"

        while true; do
            printf '\n'
            printf '%s\n' '============================================================'
            printf '%s\n' 'LUKS RECOVERY KEY'
            printf '%s\n' '============================================================'
            printf '%s\n' "$recovery_key"
            printf '%s\n' '============================================================'
            printf '\n'
            printf '%s\n' 'IMPORTANT: Save this recovery key somewhere secure.'
            printf '%s\n' 'Store it separately from your normal LUKS key.'
            printf '%s\n' 'It is not written to settings.nix or the repository.'
            printf '\n'

            read -r -p 'Have you saved the recovery key? [y/N/q] ' saved

            if [[ "$saved" =~ ^([qQ])$ ]]; then
                unset passphrase new_key recovery_key
                say 'LUKS rotation cancelled; existing key remains unchanged.'
                return 0
            fi

            if [[ "$saved" =~ ^([yY]|[yY][eE][sS])$ ]]; then
                read -r -p 'Are you absolutely sure you saved the recovery key? [y/N/q] ' saved

                if [[ "$saved" =~ ^([qQ])$ ]]; then
                    unset passphrase new_key recovery_key
                    say 'LUKS rotation cancelled; existing key remains unchanged.'
                    return 0
                fi

                if [[ "$saved" =~ ^([yY]|[yY][eE][sS])$ ]]; then
                    break
                fi
            fi

            printf '\n'
            say 'You must confirm that the recovery key has been saved.'
            say 'The key will be displayed again.'
        done

        old_file="$(mktemp)"
        new_file="$(mktemp)"
        recovery_file="$(mktemp)"

        chmod 600 "$old_file" "$new_file" "$recovery_file"

        printf '%s' "$passphrase" > "$old_file"
        printf '%s' "$new_key" > "$new_file"
        printf '%s' "$recovery_key" > "$recovery_file"

        # Ensure temporary key material is removed if the function exits
        # unexpectedly after the files have been created.
        cleanup_luks_key_files() {
            rm -f -- "$old_file" "$new_file" "$recovery_file" 2>/dev/null || true
            unset passphrase new_key recovery_key
        }

        say 'Enrolling the new LUKS key...'

        if ! sudo cryptsetup luksAddKey \
            "$device" \
            "$new_file" \
            --key-file "$old_file"; then

            cleanup_luks_key_files
            die 'Failed to enroll the new LUKS key; existing key remains valid.'
        fi

        say 'New LUKS key enrolled successfully.'

        say 'Enrolling the LUKS recovery key...'

        if ! sudo cryptsetup luksAddKey \
            "$device" \
            "$recovery_file" \
            --key-file "$old_file"; then

            cleanup_luks_key_files
            say '[ERROR] Recovery key enrollment failed.'
            say 'The new LUKS key remains enrolled.'
            say 'The original LUKS key remains valid.'
            return 1
        fi

        say 'Recovery key enrolled successfully.'

        printf '\n'
        printf '%s\n' '============================================================'
        printf '%s\n' 'Removing original LUKS key...'
        printf '%s\n' '============================================================'

        if sudo cryptsetup luksRemoveKey \
            "$device" \
            --key-file "$old_file"; then

            cleanup_luks_key_files

            printf '\n'
            printf '%s\n' '============================================================'
            printf '%s\n' 'LUKS ROTATION COMPLETE'
            printf '%s\n' '============================================================'
            printf '%s\n' 'Available unlock methods:'
            printf '%s\n' '  - New generated LUKS key'
            printf '%s\n' '  - Recovery key'
            printf '%s\n' '  - TPM2 automatic unlock (if enabled)'
            printf '\n'
            printf '%s\n' 'The original LUKS key has been removed.'
            printf '%s\n' 'The new LUKS key and recovery key remain enrolled.'
            printf '%s\n' '============================================================'
            printf '\n'

        else
            cleanup_luks_key_files

            die 'CRITICAL: New and recovery keys were enrolled, but the original key could not be removed. Check the LUKS header before continuing.'
        fi

    else
        say 'Keeping the existing LUKS key.'
        unset passphrase
    fi
}
