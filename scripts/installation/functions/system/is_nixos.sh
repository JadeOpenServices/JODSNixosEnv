is_nixos() {
    [[ -r /etc/os-release ]] && grep -qi '^ID=nixos' /etc/os-release
}
