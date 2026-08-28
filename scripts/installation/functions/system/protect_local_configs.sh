protect_local_configs() {
    local repo_root="$1"
    local hardware_file="$2"
    local git_dir exclude_file relative_hardware

    [[ -d "$repo_root/.git" ]] || {
        say '[NOTE] Git repository metadata not found; skipping local config protection.'
        return 0
    }

    command -v git >/dev/null 2>&1 || {
        say '[WARN] git is unavailable; skipping local config protection.'
        return 0
    }

    command -v realpath >/dev/null 2>&1 || {
        say '[WARN] realpath is unavailable; skipping local config protection.'
        return 0
    }

    git_dir="$(git -C "$repo_root" rev-parse --git-dir 2>/dev/null)" || {
        say '[WARN] Unable to determine Git directory; skipping local config protection.'
        return 0
    }

    if [[ "$git_dir" != /* ]]; then
        git_dir="$repo_root/$git_dir"
    fi

    exclude_file="$git_dir/info/exclude"

    mkdir -p "$(dirname "$exclude_file")"
    touch "$exclude_file"

    if ! grep -Fqx 'profiles/*/hardware-configuration.nix.bak.*' "$exclude_file"; then
        printf '%s\n' \
            'profiles/*/hardware-configuration.nix.bak.*' \
            >> "$exclude_file"
    fi

    if ! grep -Fqx 'user.config.json' "$exclude_file"; then
        printf '%s\n' 'user.config.json' >> "$exclude_file"
    fi

    # settings.nix is always machine-local.
    if [[ -f "$repo_root/settings.nix" ]]; then
        git -C "$repo_root" update-index --skip-worktree -- settings.nix
        say 'Protected local config from accidental Git commits: settings.nix'
    fi

    if [[ -f "$repo_root/user.config.json" ]] &&
       git -C "$repo_root" ls-files --error-unmatch -- user.config.json >/dev/null 2>&1; then
        git -C "$repo_root" update-index --skip-worktree -- user.config.json
        say 'Protected local config from accidental Git commits: user.config.json'
    fi

    # hardware_file is already an absolute path when supplied by the installer.
    if [[ -n "$hardware_file" && -f "$hardware_file" ]]; then
        relative_hardware="$(
            realpath --relative-to="$repo_root" "$hardware_file"
        )"

        git -C "$repo_root" update-index --skip-worktree -- "$relative_hardware"

        say "Protected local config from accidental Git commits: $relative_hardware"
    elif [[ -n "$hardware_file" ]]; then
        say "[WARN] Cannot protect missing local config: $hardware_file"
    fi

    say 'Machine-specific configuration remains available to the Nix flake.'
}
