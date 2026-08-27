configure_docker() {
    local answer
    cfg_docker_enable=false
    read -r -p 'Enable Docker daemon? Docker access is root-equivalent; rootless Podman remains available [y/N] ' answer
    [[ "$answer" =~ ^([yY]|[yY][eE][sS])$ ]] && cfg_docker_enable=true
}
