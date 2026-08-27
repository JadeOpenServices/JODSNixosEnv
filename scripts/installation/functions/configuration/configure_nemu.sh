configure_nemu() {
    local answer passthrough ids
    cfg_nemu_enable=false
    cfg_nemu_gpu_passthrough=false
    cfg_nemu_gpu_ids=()
    read -r -p 'Enable Nemu for local QEMU virtual machines? [y/N] ' answer
    [[ "$answer" =~ ^([yY]|[yY][eE][sS])$ ]] || return 0
    cfg_nemu_enable=true
    ids="${cfg_graphics_passthrough_ids:-}"
    if [[ -n "$ids" ]]; then
        read -r -p 'Pass the detected dedicated GPU through to Nemu? This removes it from the host [y/N] ' passthrough
        if [[ "$passthrough" =~ ^([yY]|[yY][eE][sS])$ ]]; then
            cfg_nemu_gpu_passthrough=true
            read -r -a cfg_nemu_gpu_ids <<< "$ids"
        fi
    fi
}
