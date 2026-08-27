detect_hardware() {
    local pci gpu wifi line vendor count bus passthrough_ids
    cfg_graphics_vendor="unknown"
    cfg_graphics_type="integrated"
    cfg_graphics_compute=false
    cfg_graphics_bus_id=""
    cfg_graphics_integrated_bus_id=""
    cfg_graphics_passthrough_ids=""
    cfg_wifi_driver=""
    command -v lspci >/dev/null 2>&1 || { say '[NOTE] lspci unavailable; using generic graphics/firmware support.'; return 0; }
    pci="$(lspci -Dn 2>/dev/null || true)"
    gpu="$(printf '%s\n' "$pci" | grep -Ei 'VGA compatible controller|3D controller|Display controller' || true)"
    count="$(printf '%s\n' "$gpu" | sed '/^$/d' | wc -l)"
    if printf '%s\n' "$gpu" | grep -qi nvidia; then cfg_graphics_vendor=nvidia; cfg_graphics_compute=true
    elif printf '%s\n' "$gpu" | grep -qi amd; then cfg_graphics_vendor=amd; cfg_graphics_compute=true
    elif printf '%s\n' "$gpu" | grep -qi intel; then cfg_graphics_vendor=intel; fi
    [[ "$count" -gt 1 ]] && cfg_graphics_type=hybrid
    [[ "$cfg_graphics_vendor" == nvidia && "$count" -eq 1 ]] && cfg_graphics_type=dedicated
    [[ "$cfg_graphics_vendor" == amd && "$count" -eq 1 ]] && cfg_graphics_type=dedicated
    bus="$(printf '%s\n' "$gpu" | head -n1 | awk '{print $1}' | awk -F'[:.]' '{printf "PCI:%d:%d:%d", $2,$3,$4}')"
    [[ "$cfg_graphics_vendor" == nvidia ]] && cfg_graphics_bus_id="$bus"
    passthrough_ids="$(printf '%s\n' "$gpu" | grep -Eiv 'Intel' | awk '{for(i=1;i<=NF;i++) if ($i ~ /^[0-9a-fA-F]{4}:[0-9a-fA-F]{4}$/) print $i}' | sort -u | paste -sd' ' -)"
    cfg_graphics_passthrough_ids="$passthrough_ids"
    if [[ "$cfg_graphics_type" == hybrid ]]; then
        cfg_graphics_integrated_bus_id="$(printf '%s\n' "$gpu" | grep -Eiv 'nvidia' | head -n1 | awk '{print $1}' | awk -F'[:.]' '{printf "PCI:%d:%d:%d", $2,$3,$4}')"
    fi
    wifi="$(printf '%s\n' "$pci" | grep -Ei 'Network controller|Wireless controller' | head -n1 || true)"
    case "$wifi" in
        *Intel*) cfg_wifi_driver=iwlwifi ;;
        *Qualcomm*|*Atheros*) cfg_wifi_driver=ath10k_pci ;;
        *MediaTek*) cfg_wifi_driver=mt7921e ;;
        *Realtek*) cfg_wifi_driver=rtw89pci ;;
        *Broadcom*) cfg_wifi_driver=brcmfmac ;;
    esac
    say "Detected graphics: ${cfg_graphics_vendor} (${cfg_graphics_type})"
    [[ -n "$cfg_wifi_driver" ]] && say "Detected Wi-Fi driver: $cfg_wifi_driver"
}
