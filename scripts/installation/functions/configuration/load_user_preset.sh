load_user_preset() {
    local repo_root preset
    repo_root="$1"
    preset="$repo_root/user.config.json"
    cfg_preset_loaded=false
    [[ -f "$preset" ]] || return 0
    grep -q '^[[:space:]]*{' "$preset" || die "Invalid JSON in $preset"
    grep -q '^[[:space:]]*}[[:space:]]*$' "$preset" || die "Invalid JSON in $preset"
    cfg_preset_loaded=true
    PRESET_FILE="$preset"
    # The supported preset is intentionally a flat JSON object. These small
    # parsers keep the installer self-contained and avoid a jq dependency.
    preset_get() {
        local key="$1"
        sed -nE "s/^[[:space:]]*\"${key}\"[[:space:]]*:[[:space:]]*\"([^\"]*)\"[[:space:]]*,?[[:space:]]*$/\1/p; s/^[[:space:]]*\"${key}\"[[:space:]]*:[[:space:]]*([^,]+),?[[:space:]]*$/\1/p" "$PRESET_FILE" | head -n1
    }
    preset_array() {
        local key="$1"
        sed -nE "s/^[[:space:]]*\"${key}\"[[:space:]]*:[[:space:]]*\[([^]]*)\].*$/\1/p" "$PRESET_FILE" | tr ',' '\n' | sed -E 's/^[[:space:]]*\"//; s/\"[[:space:]]*$//; /^[[:space:]]*$/d'
    }
    preset_bool() { [[ "$(preset_get "$1")" == true ]]; }
    say "Loaded installer preset: $preset"
}
