run_rebuild() {
    local repo_root="$1" hostname="$2"
    require_command sudo
    say "Validating the NixOS configuration before deployment..."
    sudo nixos-rebuild dry-build --flake "$repo_root#$hostname"
    say "Configuration validation passed; installing the next boot generation..."
    # `boot` writes the generation without restarting the display manager or
    # user services mid-session. The new Hyprland/SDDM stack starts on reboot.
    sudo nixos-rebuild boot --flake "$repo_root#$hostname"
}
