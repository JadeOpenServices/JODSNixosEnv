run_rebuild() {
    local repo_root="$1" hostname="$2"
    require_command sudo
    sudo nixos-rebuild switch --flake "$repo_root#$hostname"
}
