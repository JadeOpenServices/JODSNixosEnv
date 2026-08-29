configure_hardware_features() {
    local answer requested
    cfg_clamshell_enable=true
    cfg_usbguard_enable=false

    if [[ "${cfg_preset_loaded:-false}" == true ]]; then
        requested="$(preset_get clamshellEnable)"
        [[ "$requested" == false ]] && cfg_clamshell_enable=false
        requested="$(preset_get usbguardEnable)"
        [[ "$requested" == true ]] && cfg_usbguard_enable=true
        return 0
    fi

    read -r -p 'Enable clamshell mode (suspend when lid closes unless docked)? [Y/n] ' answer
    [[ "$answer" =~ ^([nN]|[nN][oO])$ ]] && cfg_clamshell_enable=false
    confirm 'Enable USBGuard? New USB devices are blocked until you permit them.' && cfg_usbguard_enable=true
}
