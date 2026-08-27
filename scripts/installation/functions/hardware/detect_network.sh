detect_network() {
    local pci wifi

    cfg_wifi_driver=""

    command -v lspci >/dev/null 2>&1 || return 0

    pci="$(lspci -Dnn 2>/dev/null || true)"

    wifi="$(
        printf '%s\n' "$pci" |
            grep -Ei 'Network controller|Wireless controller' |
            head -n1 ||
            true
    )"

    case "$wifi" in
        *Intel*)
            cfg_wifi_driver="iwlwifi"
            ;;
        *Qualcomm*|*Atheros*)
            cfg_wifi_driver="ath10k_pci"
            ;;
        *MediaTek*)
            cfg_wifi_driver="mt7921e"
            ;;
        *Realtek*)
            cfg_wifi_driver="rtw89pci"
            ;;
        *Broadcom*)
            cfg_wifi_driver="brcmfmac"
            ;;
    esac

    if [[ -n "$cfg_wifi_driver" ]]; then
        say "Detected Wi-Fi driver: $cfg_wifi_driver"
    else
        say "Detected Wi-Fi driver: unknown"
    fi

    return 0
}