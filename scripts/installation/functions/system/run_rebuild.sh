run_rebuild() {
    local repo_root="$1" hostname="$2"
    require_command sudo
    say "Validating the NixOS configuration before deployment..."
    sudo nixos-rebuild dry-build --flake "$repo_root#$hostname"
    say "Configuration validation passed; applying the new system generation..."
    sudo nixos-rebuild switch --flake "$repo_root#$hostname"
}
