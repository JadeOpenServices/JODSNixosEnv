configure_tpm2_luks() {
    local hardware_file="$1"
    local device=""
    local name=""
    local answer=""
    local passphrase=""
    local key_file=""

    [[ -f "$hardware_file" ]] || return 0

    name="$(
        sed -n \
            's/.*boot\.initrd\.luks\.devices\.\?"\{0,1\}\([^".]*\)"\{0,1\}\.device.*/\1/p' \
            "$hardware_file" |
            head -n1
    )"

    [[ -n "$name" ]] || {
        say '[NOTE] No LUKS device mapping found for TPM2 enrollment.'
        return 0
    }

    if [[ ! -e /dev/tpmrm0 && ! -e /dev/tpm0 ]]; then
        say '[NOTE] No TPM2 device detected; keeping passphrase unlock.'
        return 0
    fi

    command -v systemd-cryptenroll >/dev/null 2>&1 || {
        say '[WARN] systemd-cryptenroll is unavailable; skipping TPM2 enrollment.'
        return 0
    }

    device="$(
        lsblk -pnro NAME,FSTYPE 2>/dev/null |
            awk '$2 == "crypto_LUKS" {print $1; exit}'
    )"

    [[ -n "$device" ]] || {
        say '[NOTE] No active crypto_LUKS device found; skipping TPM2 enrollment.'
        return 0
    }

    printf '\n'
    read -r -p \
        'Enable TPM2 automatic unlock for this LUKS device? Keep the passphrase as recovery [y/N] ' \
        answer

    [[ "$answer" =~ ^([yY]|[yY][eE][sS])$ ]] || {
        say 'TPM2 automatic unlock not enabled.'
        return 0
    }

    printf '\n'
    read -r -s -p \
        "Enter the current LUKS passphrase for TPM enrollment: " \
        passphrase
    printf '\n'

    [[ -n "$passphrase" ]] || {
        say '[WARN] No passphrase supplied; skipping TPM2 enrollment.'
        return 0
    }

    key_file="$(mktemp)"
    chmod 600 "$key_file"
    printf '%s' "$passphrase" > "$key_file"

    if sudo systemd-cryptenroll \
        --unlock-key-file="$key_file" \
        --tpm2-device=auto \
        "$device"; then

        sed -i '$i\
    boot.initrd.systemd.enable = true;\
    boot.initrd.systemd.tpm2.enable = true;\
    boot.initrd.luks.devices."'"$name"'".crypttabExtraOpts = [ "tpm2-device=auto" ];' \
            "$hardware_file"

        cfg_luks_tpm2_enable=true

        say 'TPM2 unlock enrolled successfully.'
        say 'The existing LUKS passphrase remains available as recovery.'
    else
        say '[ERROR] TPM2 enrollment failed; no boot configuration was changed.' >&2
    fi

    rm -f -- "$key_file"
    unset passphrase

    return 0
}