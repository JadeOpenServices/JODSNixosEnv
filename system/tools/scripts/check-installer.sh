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

    printf '\n%s\n' '========== 2. GRAPHICS DETECTOR =========='

    if bash -n "$detector"; then
        printf '%s\n' 'OK: detect_graphics.sh syntax'
    else
        printf '%s\n' 'ERROR: detect_graphics.sh syntax'
        return 1
    fi

    printf '\n%s\n' '========== 3. FRESH HARDWARE DETECTION =========='

    (
        cd "$repo_root" || exit 1
        source "$detector"

        say() {
            printf '%s\n' "$*"
        }

        detect_graphics

        printf '%s\n' ''
        printf '%s\n' '========== 4. DETECTED VARIABLES =========='
        printf 'vendor=%s\n' "$cfg_graphics_vendor"
        printf 'type=%s\n' "$cfg_graphics_type"
        printf 'compute=%s\n' "$cfg_graphics_compute"
        printf 'bus=%s\n' "$cfg_graphics_bus_id"
        printf 'integrated_bus=%s\n' "$cfg_graphics_integrated_bus_id"
        printf 'passthrough_ids=%s\n' "$cfg_graphics_passthrough_ids"
    )

    printf '\n%s\n' '========== 5. INSTALLER ORDER =========='
    grep -nE \
        'configure_ai|detect_graphics|configure_nemu|render_settings' \
        "$installer" || true

    printf '\n%s\n' '========== 6. RENDER OUTPUT REFERENCES =========='
    grep -nE \
        'graphicsVendor|graphicsType|graphicsCompute|graphicsBusId|graphicsIntegratedBusId|aiEnable|aiModel|aiContextTokens|aiVramMB' \
        "$renderer" || true

    printf '\n%s\n' '========== 7. OLLAMA =========='
    grep -nE \
        'graphicsVendor|graphicsType|graphicsIntegratedBusId|OLLAMA_IGPU_ENABLE|OLLAMA_CONTEXT_LENGTH|settings.aiModel' \
        "$ollama" || true

    printf '\n%s\n' '========== 8. CURRENT GENERATED SETTINGS =========='

    if [[ -f "$settings" ]]; then
        grep -nE \
            'graphicsVendor|graphicsType|graphicsCompute|graphicsBusId|graphicsIntegratedBusId|aiEnable|aiModel|aiContextTokens|aiVramMB' \
            "$settings" || true
    else
        printf '%s\n' 'NOTE: settings.nix does not exist yet'
    fi

    printf '\n%s\n' '========== DONE =========='
}

check_installer
