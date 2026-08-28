configure_nemu() {
    local answer passthrough ids
    cfg_nemu_enable=false
    cfg_nemu_gpu_passthrough=false
    cfg_nemu_gpu_ids=()
    if [[ "${cfg_preset_loaded:-false}" == true ]]; then
        cfg_nemu_enable="$(preset_bool nemuEnable && echo true || echo false)"
        cfg_nemu_gpu_passthrough="$(preset_bool nemuGpuPassthrough && echo true || echo false)"
        # PCI IDs are always detected from the current machine; never trust
        # stale/manual IDs from a portable preset.
        cfg_nemu_gpu_ids=()
        read -r -a cfg_nemu_gpu_ids <<< "${cfg_graphics_passthrough_ids:-}"
        [[ ${#cfg_nemu_gpu_ids[@]} -gt 0 ]] || cfg_nemu_gpu_passthrough=false
        return 0
    fi
    if [[ "${INSTALLER_UI:-terminal}" == gtk ]]; then
        confirm 'Enable Nemu for local QEMU virtual machines?' && answer=y || answer=n
    else
        read -r -p 'Enable Nemu for local QEMU virtual machines? [y/N] ' answer
    fi
    [[ "$answer" =~ ^([yY]|[yY][eE][sS])$ ]] || return 0
    cfg_nemu_enable=true
    ids="${cfg_graphics_passthrough_ids:-}"
    if [[ -n "$ids" ]]; then
        if [[ "${INSTALLER_UI:-terminal}" == gtk ]]; then
            confirm 'Pass the detected dedicated GPU through to Nemu? This removes it from the host.' && passthrough=y || passthrough=n
        else
            read -r -p 'Pass the detected dedicated GPU through to Nemu? This removes it from the host [y/N] ' passthrough
        fi
        if [[ "$passthrough" =~ ^([yY]|[yY][eE][sS])$ ]]; then
            cfg_nemu_gpu_passthrough=true
            read -r -a cfg_nemu_gpu_ids <<< "$ids"
        fi
    fi
}
