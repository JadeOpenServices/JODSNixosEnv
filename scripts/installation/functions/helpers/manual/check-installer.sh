#!/usr/bin/env bash

set -u

REPO_ROOT="${REPO_ROOT:-$HOME/Documents/gjallarOS}"

check_installer() {
    local repo_root="$REPO_ROOT"
    local detector="$repo_root/scripts/installation/functions/hardware/detect_graphics.sh"
    local renderer="$repo_root/scripts/installation/functions/configuration/render_settings.sh"
    local installer="$repo_root/scripts/installation/install.sh"
    local ollama="$repo_root/system/apps/ollama.nix"
    local settings="$repo_root/settings.nix"
    local user_config="$repo_root/user.config.json"
    local preset="$repo_root/scripts/installation/user_PresetJSON/default.user.config.json"

    cd "$repo_root" || {
        printf '%s\n' "ERROR: cannot enter repository: $repo_root"
        return 1
    }

    printf '%s\n' '========== 1. ROOT CONFIG =========='

    if nix-instantiate --eval --expr \
        "builtins.fromJSON (builtins.readFile ./user.config.json)" \
        >/dev/null 2>&1; then
        printf '%s\n' 'OK: user.config.json is valid JSON'
    else
        printf '%s\n' 'ERROR: user.config.json is INVALID JSON'
        nix-instantiate --eval --expr \
            "builtins.fromJSON (builtins.readFile ./user.config.json)" \
            2>&1
        return 1
    fi

    printf '\n%s\n' '========== 2. PRESET KEY CHECK =========='

    if [[ ! -f "$preset" ]]; then
        printf '%s\n' "ERROR: default preset missing: $preset"
        return 1
    fi

    if [[ ! -f "$user_config" ]]; then
        printf '%s\n' "ERROR: user.config.json missing: $user_config"
        return 1
    fi

    local key
    local missing=0

    while IFS= read -r key; do
        [[ -z "$key" ]] && continue

        if grep -q "\"${key}\"[[:space:]]*:" "$user_config"; then
            printf 'OK: %s\n' "$key"
        else
            printf 'MISSING: %s\n' "$key"
            missing=1
        fi
    done < <(
        grep -oE '"[A-Za-z_][A-Za-z0-9_]*"[[:space:]]*:' "$preset" |
            sed -E 's/^"([^"]+)".*/\1/' |
            sort -u
    )

    if (( missing != 0 )); then
        printf '%s\n' 'ERROR: user.config.json is missing preset keys'
        return 1
    fi

    printf '%s\n' 'OK: user.config.json contains all preset keys'

    printf '\n%s\n' '========== 3. GRAPHICS DETECTOR =========='

    if bash -n "$detector"; then
        printf '%s\n' 'OK: detect_graphics.sh syntax'
    else
        printf '%s\n' 'ERROR: detect_graphics.sh syntax'
        return 1
    fi

    printf '\n%s\n' '========== 4. FRESH HARDWARE DETECTION =========='

    source "$detector"

    say() {
        printf '%s\n' "$*"
    }

    detect_graphics

    printf '\n%s\n' '========== 5. DETECTED VARIABLES =========='
    printf 'vendor=%s\n' "${cfg_graphics_vendor:-}"
    printf 'type=%s\n' "${cfg_graphics_type:-}"
    printf 'compute=%s\n' "${cfg_graphics_compute:-}"
    printf 'bus=%s\n' "${cfg_graphics_bus_id:-}"
    printf 'integrated_bus=%s\n' "${cfg_graphics_integrated_bus_id:-}"
    printf 'passthrough_ids=%s\n' "${cfg_graphics_passthrough_ids:-}"

    printf '\n%s\n' '========== 6. INSTALLER ORDER =========='

    grep -nE \
        'configure_ai|detect_graphics|configure_nemu|render_settings' \
        "$installer" || true

    printf '\n%s\n' '========== 7. RENDER OUTPUT REFERENCES =========='

    grep -nE \
        'graphicsVendor|graphicsType|graphicsCompute|graphicsBusId|graphicsIntegratedBusId|aiEnable|aiModel|aiContextTokens|aiVramMB' \
        "$renderer" || true

    printf '\n%s\n' '========== 8. OLLAMA =========='

    grep -nE \
        'graphicsVendor|graphicsType|graphicsIntegratedBusId|OLLAMA_IGPU_ENABLE|OLLAMA_CONTEXT_LENGTH|settings.aiModel' \
        "$ollama" || true

    printf '\n%s\n' '========== 9. CURRENT GENERATED SETTINGS =========='

    if [[ -f "$settings" ]]; then
        grep -nE \
            'graphicsVendor|graphicsType|graphicsCompute|graphicsBusId|graphicsIntegratedBusId|aiEnable|aiModel|aiContextTokens|aiVramMB' \
            "$settings" || true
    else
        printf '%s\n' 'NOTE: settings.nix does not exist yet'
    fi

    printf '\n%s\n' '========== 10. LIVE VS GENERATED GRAPHICS =========='

    if [[ -f "$settings" ]]; then
        local generated_vendor generated_type generated_compute
        local generated_bus generated_integrated

        generated_vendor="$(
            sed -n 's/^[[:space:]]*graphicsVendor = "\([^"]*\)".*/\1/p' "$settings" |
                head -n1
        )"

        generated_type="$(
            sed -n 's/^[[:space:]]*graphicsType = "\([^"]*\)".*/\1/p' "$settings" |
                head -n1
        )"

        generated_compute="$(
            sed -n 's/^[[:space:]]*graphicsCompute = \(true\|false\);.*/\1/p' "$settings" |
                head -n1
        )"

        generated_bus="$(
            sed -n 's/^[[:space:]]*graphicsBusId = "\([^"]*\)".*/\1/p' "$settings" |
                head -n1
        )"

        generated_integrated="$(
            sed -n 's/^[[:space:]]*graphicsIntegratedBusId = "\([^"]*\)".*/\1/p' "$settings" |
                head -n1
        )"

        printf 'live vendor:             %s\n' "${cfg_graphics_vendor:-}"
        printf 'generated vendor:        %s\n' "$generated_vendor"
        printf 'live type:               %s\n' "${cfg_graphics_type:-}"
        printf 'generated type:          %s\n' "$generated_type"
        printf 'live compute:            %s\n' "${cfg_graphics_compute:-}"
        printf 'generated compute:       %s\n' "$generated_compute"
        printf 'live bus:                %s\n' "${cfg_graphics_bus_id:-}"
        printf 'generated bus:           %s\n' "$generated_bus"
        printf 'live integrated bus:     %s\n' "${cfg_graphics_integrated_bus_id:-}"
        printf 'generated integrated:    %s\n' "$generated_integrated"

        if [[ "${cfg_graphics_vendor:-}" == "$generated_vendor" &&
              "${cfg_graphics_type:-}" == "$generated_type" &&
              "${cfg_graphics_compute:-}" == "$generated_compute" &&
              "${cfg_graphics_bus_id:-}" == "$generated_bus" &&
              "${cfg_graphics_integrated_bus_id:-}" == "$generated_integrated" ]]; then
            printf '%s\n' 'OK: generated graphics settings match fresh detection'
        else
            printf '%s\n' 'WARNING: generated graphics settings differ from fresh detection'
            printf '%s\n' 'NOTE: run the installer/configuration step to regenerate settings.nix'
        fi
    fi

    printf '\n%s\n' '========== 11. HELPER REGISTRATION =========='

    if [[ -f "$repo_root/system/tools/scripts/default.nix" ]] &&
       grep -q 'check-installer' "$repo_root/system/tools/scripts/default.nix"; then
        printf '%s\n' 'OK: check-installer registered in system/tools/scripts/default.nix'
    else
        printf '%s\n' 'WARNING: check-installer missing from system/tools/scripts/default.nix'
    fi

    if [[ -f "$repo_root/system/tools/scripts/help.sh" ]] &&
       grep -q 'check-installer' "$repo_root/system/tools/scripts/help.sh"; then
        printf '%s\n' 'OK: check-installer registered in help menu'
    else
        printf '%s\n' 'WARNING: check-installer missing from help.sh'
    fi

    printf '\n%s\n' '========== DONE =========='
}

check_installer
