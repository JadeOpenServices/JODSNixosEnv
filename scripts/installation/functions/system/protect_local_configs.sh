protect_local_configs() {
    local repo_root="$1"
    local profile="$2"
    local settings_file hardware_file

    settings_file="$repo_root/settings.nix"
    hardware_file="$repo_root/profiles/$profile/hardware-configuration.nix"

    command -v git >/dev/null 2>&1 || {
        say '[WARN] git is unavailable; local configs will remain visible to Git.'
        return 0
    }

    [[ -d "$repo_root/.git" ]] || {
        say '[WARN] Repository metadata not found; local configs will remain visible to Git.'
        return 0
    }

    protect_file() {
        local file="$1"
        local relative

        relative="${file#"$repo_root"/}"

        [[ -f "$file" ]] || {
            say "[WARN] Cannot protect missing local config: $relative"
            return 0
        }

        # The file must already be tracked. This is intentional:
        # tracked files remain visible to the Nix flake while their local
        # modifications are hidden from normal Git status/commits.
        if ! git -C "$repo_root" ls-files --error-unmatch -- "$relative" \
            >/dev/null 2>&1; then
            say "[WARN] $relative is not tracked by Git; cannot use skip-worktree."
            say "[WARN] Leaving it visible to Git."
            return 0
        fi

        # Clear assume-unchanged first in case another Git mechanism
        # previously marked the file.
        git -C "$repo_root" update-index \
            --no-assume-unchanged \
            --skip-worktree \
            -- "$relative"

        say "Protected local config from accidental Git commits: $relative"
    }

    say 'Protecting machine-specific configuration from accidental commits...'

    protect_file "$settings_file"
    protect_file "$hardware_file"

    say 'Machine-specific configuration remains available to the Nix flake.'
}
