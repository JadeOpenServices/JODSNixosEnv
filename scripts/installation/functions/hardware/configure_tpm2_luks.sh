configure_tpm2_luks() {
    local hardware_file="$1" device name answer passphrase key_file
    [[ -f "$hardware_file" ]] || return 0
    name="$(sed -n 's/.*boot\.initrd\.luks\.devices\.\?"\{0,1\}\([^".]*\)"\{0,1\}\.device.*/\1/p' "$hardware_file" | head -n1)"
    [[ -n "$name" ]] || return 0
    [[ -e /dev/tpmrm0 || -e /dev/tpm0 ]] || { say '[NOTE] No TPM2 device detected; keeping passphrase unlock.'; return 0; }
    command -v systemd-cryptenroll >/dev/null 2>&1 || { say '[WARN] systemd-cryptenroll is unavailable; skipping TPM2 enrollment.'; return 0; }
    device="$(lsblk -pnro NAME,FSTYPE 2>/dev/null | awk '$2 == "crypto_LUKS" {print $1; exit}')"
    [[ -n "$device" ]] || return 0
    read -r -p 'Enable TPM2 automatic unlock for this LUKS device? Keep the passphrase as recovery [y/N] ' answer
    [[ "$answer" =~ ^([yY]|[yY][eE][sS])$ ]] || return 0
    read -r -s -p "Enter the current LUKS passphrase for TPM enrollment: " passphrase; printf '\n'
    [[ -n "$passphrase" ]] || return 0
    key_file="$(mktemp)"; chmod 600 "$key_file"; printf '%s' "$passphrase" > "$key_file"
    if sudo systemd-cryptenroll --unlock-key-file="$key_file" --tpm2-device=auto "$device"; then
        sed -i '$i\\    boot.initrd.systemd.enable = true;\n    boot.initrd.systemd.tpm2.enable = true;\n    boot.initrd.luks.devices."'"$name"'".crypttabExtraOpts = [ "tpm2-device=auto" ];' "$hardware_file"
        cfg_luks_tpm2_enable=true
        say 'TPM2 unlock enrolled; the passphrase remains available as recovery.'
    else
        say 'TPM2 enrollment failed; no boot configuration was changed.' >&2
    fi
    rm -f -- "$key_file"; unset passphrase
}
