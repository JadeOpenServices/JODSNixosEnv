configure_docker() {
    local answer
    cfg_docker_enable=false
    [[ "${cfg_preset_loaded:-false}" == true ]] && { cfg_docker_enable="$(preset_bool dockerEnable && echo true || echo false)"; return 0; }
    read -r -p 'Enable Docker daemon? Docker access is root-equivalent; rootless Podman remains available [y/N] ' answer
    [[ "$answer" =~ ^([yY]|[yY][eE][sS])$ ]] && cfg_docker_enable=true
}
