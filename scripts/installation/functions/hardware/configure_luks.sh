configure_luks() {
    local hardware_file="$1"

    local device=""
    local passphrase=""
    local change=""
    local saved=""
    local confirm_key=""
    local recovery_saved=""
    local recovery_confirm=""

    local new_key=""
    local recovery_key=""

    local old_file=""
    local new_file=""
    local recovery_file=""

    [[ -f "$hardware_file" ]] || return 0

    grep -qE 'boot\.initrd\.luks\.devices|luks\.devices' "$hardware_file" || {
        return 0
    }

    say '[NOTE] LUKS mapping detected in the generated hardware configuration.'

    command -v cryptsetup >/dev/null 2>&1 || {
        say '[WARN] cryptsetup is unavailable; skipping LUKS key configuration.'
        return 0
    }

    command -v sudo >/dev/null 2>&1 || {
        say '[WARN] sudo is unavailable; skipping LUKS key configuration.'
        return 0
    }

    command -v od >/dev/null 2>&1 || {
        say '[WARN] od is unavailable; skipping LUKS key configuration.'
        return 0
    }

    device="$(
        lsblk -pnro NAME,FSTYPE 2>/dev/null |
            awk '$2 == "crypto_LUKS" {print $1; exit}'
    )"

    [[ -n "$device" ]] || {
        say '[WARN] No active crypto_LUKS device was found; skipping LUKS key configuration.'
        return 0
    }

    # -----------------------------------------------------------------------
    # 1. Obtain and verify the current LUKS key.
    # -----------------------------------------------------------------------

    printf '\n'
    printf 'LUKS device: %s\n' "$device"
    printf '\n'

    read -r -s -p \
        "Enter the current LUKS key for $device (empty to skip): " \
        passphrase
    printf '\n'

    if [[ -z "$passphrase" ]]; then
        say 'No LUKS key supplied; keeping the existing configuration.'
        return 0
    fi

    old_file="$(mktemp)"
    chmod 600 "$old_file"
    printf '%s' "$passphrase" > "$old_file"

    if ! sudo cryptsetup open \
        --test-passphrase \
        --type luks \
        "$device" \
        --key-file "$old_file" \
        2>/dev/null; then

        rm -f -- "$old_file"
        unset passphrase

        die 'The supplied LUKS key was rejected; no disk changes were made.'
    fi

    say 'Current LUKS key verified successfully.'

    # -----------------------------------------------------------------------
    # 2. Ask whether the current key should be replaced.
    # -----------------------------------------------------------------------

    printf '\n'

    read -r -p \
        'Generate a new random LUKS key and replace the current key? [y/N] ' \
        change

    if [[ ! "$change" =~ ^([yY]|[yY][eE][sS])$ ]]; then
        say 'Keeping the existing LUKS key unchanged.'

        rm -f -- "$old_file"
        unset passphrase

        return 0
    fi

    # -----------------------------------------------------------------------
    # 3. Generate the new 256-bit LUKS key.
    # -----------------------------------------------------------------------

    new_key="$(
        od -An -N32 -tx1 /dev/urandom |
            tr -d '[:space:]'
    )"

    if [[ ${#new_key} -ne 64 ]]; then
        rm -f -- "$old_file"
        unset passphrase new_key

        die 'Failed to generate a secure random LUKS key.'
    fi

    # -----------------------------------------------------------------------
    # 4. Display the new key until the user confirms it was saved.
    # -----------------------------------------------------------------------

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

        read -r -p \
            'Have you saved the new LUKS key? [y/N/q] ' \
            saved

        case "$saved" in
            y|Y|yes|YES|Yes)
                break
                ;;

            q|Q|quit|QUIT|Quit)
                say 'LUKS key generation cancelled.'
                say 'No LUKS keys were changed.'

                rm -f -- "$old_file"
                unset passphrase new_key

                return 0
                ;;

            *)
                say 'The new LUKS key must be saved before continuing.'
                say 'Displaying it again.'
                ;;
        esac
    done

    # -----------------------------------------------------------------------
    # 5. Second confirmation for the new key.
    # -----------------------------------------------------------------------

    while true; do
        read -r -p \
            'Are you absolutely sure you saved the new LUKS key? [y/N/q] ' \
            confirm_key

        case "$confirm_key" in
            y|Y|yes|YES|Yes)
                break
                ;;

            q|Q|quit|QUIT|Quit)
                say 'LUKS key generation cancelled.'
                say 'No LUKS keys were changed.'

                rm -f -- "$old_file"
                unset passphrase new_key

                return 0
                ;;

            *)
                say 'Confirmation required.'
                say 'The new LUKS key has NOT been enrolled.'
                ;;
        esac
    done

    # -----------------------------------------------------------------------
    # 6. Enroll the new key.
    #
    # The old key remains valid at this point.
    # -----------------------------------------------------------------------

    new_file="$(mktemp)"
    chmod 600 "$new_file"
    printf '%s' "$new_key" > "$new_file"

    say 'Enrolling the new LUKS key...'

    if ! sudo cryptsetup luksAddKey \
        "$device" \
        "$new_file" \
        --key-file "$old_file"; then

        rm -f -- "$old_file" "$new_file"
        unset passphrase new_key

        die 'Failed to enroll the new LUKS key; the existing key remains valid.'
    fi

    rm -f -- "$new_file"
    new_file=""

    say 'New LUKS key enrolled successfully.'
    say 'The original LUKS key remains valid until the final rotation step.'

    # -----------------------------------------------------------------------
    # 7. Generate a separate recovery key.
    # -----------------------------------------------------------------------

    recovery_key="$(
        od -An -N32 -tx1 /dev/urandom |
            tr -d '[:space:]'
    )"

    if [[ ${#recovery_key} -ne 64 ]]; then
        rm -f -- "$old_file"
        unset passphrase new_key recovery_key

        die 'Failed to generate a secure random recovery key.'
    fi

    # -----------------------------------------------------------------------
    # 8. Display recovery key until saved.
    # -----------------------------------------------------------------------

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

        read -r -p \
            'Have you saved the recovery key? [y/N/q] ' \
            recovery_saved

        case "$recovery_saved" in
            y|Y|yes|YES|Yes)
                break
                ;;

            q|Q|quit|QUIT|Quit)
                printf '\n'
                say 'WARNING: The new LUKS key has already been enrolled.'
                say 'The recovery key has NOT been enrolled.'
                printf '\n'

                while true; do
                    read -r -p \
                        'Exit recovery-key setup anyway? [y/N] ' \
                        recovery_confirm

                    case "$recovery_confirm" in
                        y|Y|yes|YES|Yes)
                            say 'Leaving recovery-key setup.'
                            say 'The original LUKS key remains valid.'
                            say 'The newly generated LUKS key remains valid.'

                            rm -f -- "$old_file"
                            unset passphrase new_key recovery_key

                            return 0
                            ;;

                        *)
                            say 'Recovery-key setup remains active.'
                            ;;
                    esac
                done
                ;;

            *)
                say 'The recovery key must be saved before continuing.'
                say 'Displaying it again.'
                ;;
        esac
    done

    # -----------------------------------------------------------------------
    # 9. Second confirmation for the recovery key.
    # -----------------------------------------------------------------------

    while true; do
        read -r -p \
            'Are you absolutely sure you saved the recovery key? [y/N/q] ' \
            recovery_confirm

        case "$recovery_confirm" in
            y|Y|yes|YES|Yes)
                break
                ;;

            q|Q|quit|QUIT|Quit)
                printf '\n'
                say 'WARNING: The new LUKS key has already been enrolled.'
                say 'The recovery key has NOT been enrolled.'
                printf '\n'

                while true; do
                    read -r -p \
                        'Exit recovery-key setup anyway? [y/N] ' \
                        recovery_confirm

                    case "$recovery_confirm" in
                        y|Y|yes|YES|Yes)
                            say 'Leaving recovery-key setup.'
                            say 'The original LUKS key remains valid.'
                            say 'The newly generated LUKS key remains valid.'

                            rm -f -- "$old_file"
                            unset passphrase new_key recovery_key

                            return 0
                            ;;

                        *)
                            say 'Recovery-key setup remains active.'
                            ;;
                    esac
                done
                ;;

            *)
                say 'Confirmation required.'
                say 'The recovery key has NOT been enrolled.'
                ;;
        esac
    done

    # -----------------------------------------------------------------------
    # 10. Enroll the recovery key.
    #
    # At this point both replacement credentials are ready to be enrolled.
    # -----------------------------------------------------------------------

    recovery_file="$(mktemp)"
    chmod 600 "$recovery_file"
    printf '%s' "$recovery_key" > "$recovery_file"

    say 'Enrolling the LUKS recovery key...'

    if ! sudo cryptsetup luksAddKey \
        "$device" \
        "$recovery_file" \
        --key-file "$old_file"; then

        rm -f -- "$old_file" "$recovery_file"
        unset passphrase new_key recovery_key

        die 'Failed to enroll the recovery key; the original and new LUKS keys remain valid.'
    fi

    say 'Recovery key enrolled successfully.'

    # -----------------------------------------------------------------------
    # 11. Both replacement keys are now enrolled.
    #
    # Only now is it safe to remove the original key.
    # -----------------------------------------------------------------------

    printf '\n'
    say 'Both the new LUKS key and recovery key are enrolled.'
    say 'The original LUKS key will now be removed.'

    if ! sudo cryptsetup luksRemoveKey \
        "$device" \
        "$old_file"; then

        rm -f -- "$old_file" "$recovery_file"
        unset passphrase new_key recovery_key

        die 'Failed to remove the original LUKS key. The new and recovery keys remain valid.'
    fi

    say 'Original LUKS key removed successfully.'

    # -----------------------------------------------------------------------
    # 12. Verify that the old key can no longer unlock the device.
    # -----------------------------------------------------------------------

    if sudo cryptsetup open \
        --test-passphrase \
        --type luks \
        "$device" \
        --key-file "$old_file" \
        2>/dev/null; then

        rm -f -- "$old_file" "$recovery_file"
        unset passphrase new_key recovery_key

        die 'SECURITY CHECK FAILED: the original LUKS key still appears to be valid.'
    fi

    say 'Verified: the original LUKS key is no longer accepted.'

    # -----------------------------------------------------------------------
    # 13. Final state.
    # -----------------------------------------------------------------------

    printf '\n'
    printf '%s\n' '============================================================'
    printf '%s\n' 'LUKS KEY ROTATION COMPLETE'
    printf '%s\n' '============================================================'
    printf '%s\n' 'Available unlock methods:'
    printf '%s\n' '  - New generated LUKS key'
    printf '%s\n' '  - Recovery key'

    if [[ "${cfg_luks_tpm2_enable:-false}" == true ]]; then
        printf '%s\n' '  - TPM2 automatic unlock'
    fi

    printf '%s\n' '============================================================'
    printf '\n'

    say 'The original LUKS key has been removed.'
    say 'The new LUKS key and recovery key remain enrolled.'

    # -----------------------------------------------------------------------
    # 14. Remove temporary sensitive material.
    # -----------------------------------------------------------------------

    rm -f -- "$old_file" "$new_file" "$recovery_file"

    unset passphrase
    unset new_key
    unset recovery_key

    return 0
}