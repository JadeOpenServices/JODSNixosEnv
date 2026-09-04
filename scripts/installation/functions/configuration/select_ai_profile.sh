#!/usr/bin/env bash

# The selection policy lives in internal/ai/profile. This adapter only passes
# its validated output to the existing installer variables.
select_ai_profile() {
    local script_dir repo_root profile_output key value
    local profile='' model='' context_tokens='' vram_mb='' ram_gb='' gpu_vendor='' gpu_type=''

    script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    repo_root="$(cd "$script_dir/../../../.." && pwd)"

    if command -v gjallarctl >/dev/null 2>&1; then
        profile_output="$(gjallarctl ai profile --config "$repo_root/user.config.json")" || {
            say 'ERROR: Go AI profile selection failed.'
            return 1
        }
    else
        # Fresh NixOS hosts may not have gjallarctl installed yet. Nix builds
        # the same pinned local tool without falling back to unsafe shell eval.
        profile_output="$(
            nix --extra-experimental-features 'nix-command flakes' run \
                "path:$repo_root#gjallarctl" -- ai profile \
                --config "$repo_root/user.config.json"
        )" || {
            say 'ERROR: Could not build or run the Go AI profile selector.'
            return 1
        }
    fi

    while IFS='=' read -r key value; do
        case "$key" in
            profile) profile="$value" ;;
            model) model="$value" ;;
            context_tokens) context_tokens="$value" ;;
            vram_mb) vram_mb="$value" ;;
            ram_gb) ram_gb="$value" ;;
            gpu_vendor) gpu_vendor="$value" ;;
            gpu_type) gpu_type="$value" ;;
            *)
                say "ERROR: Go AI selector returned unknown field: $key"
                return 1
                ;;
        esac
    done <<< "$profile_output"

    [[ "$profile" =~ ^(low-memory|integrated|dedicated|user-override)$ ]] || {
        say 'ERROR: Go AI selector returned an invalid profile.'
        return 1
    }
    [[ "$context_tokens" =~ ^[0-9]+$ && "$vram_mb" =~ ^[0-9]+$ && "$ram_gb" =~ ^[0-9]+$ ]] || {
        say 'ERROR: Go AI selector returned invalid numeric values.'
        return 1
    }
    [[ -n "$model" && "$model" != *$'\n'* && "$model" != *$'\r'* ]] || {
        say 'ERROR: Go AI selector returned an invalid model name.'
        return 1
    }

    cfg_ai_profile="$profile"
    cfg_ai_model="$model"
    cfg_ai_context_tokens="$context_tokens"
    cfg_ai_vram_mb="$vram_mb"

    say 'AI hardware evaluation:'
    printf '  RAM:     %s GB\n' "$ram_gb"
    printf '  GPU:     %s\n' "$gpu_vendor"
    printf '  Type:    %s\n' "$gpu_type"
    printf '  VRAM:    %s MB\n' "$cfg_ai_vram_mb"
    printf '  Profile: %s\n' "$cfg_ai_profile"
    printf '  Model:   %s\n' "$cfg_ai_model"
    printf '  Context: %s\n' "$cfg_ai_context_tokens"
}
