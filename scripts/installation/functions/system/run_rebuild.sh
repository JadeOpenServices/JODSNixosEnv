run_rebuild() {
    local script_dir repo_root hostname

    require_command sudo
    require_command nixos-rebuild

    # Resolve the directory containing this script/function.
    # BASH_SOURCE[0] points to the file where run_rebuild is defined.
    script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)" || {
        say "Error: could not determine the GjallarOS script directory."
        return 1
    }

    # Walk upward until we find the flake root.
    repo_root="$script_dir"
    while [[ "$repo_root" != "/" && ! -f "$repo_root/flake.nix" ]]; do
        repo_root="$(dirname -- "$repo_root")"
    done

    if [[ ! -f "$repo_root/flake.nix" ]]; then
        say "Error: could not locate the GjallarOS flake root."
        return 1
    fi

    # The NixOS configuration name is the system hostname.
    hostname="$(hostname)"

    say "Using GjallarOS flake: $repo_root"
    say "Target configuration: $hostname"
    say "Validating the NixOS configuration before deployment..."

    sudo nixos-rebuild dry-build \
        --flake "$repo_root#$hostname" \
        --show-trace || return 1

    say "Configuration validation passed; installing the next boot generation..."

    # 'boot' writes the generation without restarting the display manager
    # or user services mid-session. The new Hyprland/SDDM stack starts on reboot.
    sudo nixos-rebuild boot \
        --flake "$repo_root#$hostname"
}

alias rebuild='run_rebuild'