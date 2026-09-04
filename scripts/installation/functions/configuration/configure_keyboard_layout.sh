#!/usr/bin/env bash

# The input normalization policy lives in internal/input/xkb.
configure_keyboard_layout() {
    local requested normalized key value script_dir repo_root

    if [[ "${cfg_preset_loaded:-false}" == true ]]; then
        requested="$(preset_get keyboardLayout)"
    else
        prompt_value 'Keyboard layout (for example: de or us):' 'de'
        requested="$REPLY"
    fi

    script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    repo_root="$(cd "$script_dir/../../../.." && pwd)"

    if command -v gjallarctl >/dev/null 2>&1; then
        normalized="$(gjallarctl normalize keyboard --layout "$requested")" ||
            die 'Invalid keyboard layout.'
    else
        normalized="$(
            nix --extra-experimental-features 'nix-command flakes' run \
                "path:$repo_root#gjallarctl" -- normalize keyboard --layout "$requested"
        )" || die 'Invalid keyboard layout.'
    fi

    cfg_keyboard_layout=''
    cfg_keyboard_variant=''
    while IFS='=' read -r key value; do
        case "$key" in
            layout) cfg_keyboard_layout="$value" ;;
            variant) cfg_keyboard_variant="$value" ;;
            *) die "Go keyboard normalizer returned unknown field: $key" ;;
        esac
    done <<< "$normalized"
    [[ -n "$cfg_keyboard_layout" ]] || die 'Go keyboard normalizer returned no layout.'
}
