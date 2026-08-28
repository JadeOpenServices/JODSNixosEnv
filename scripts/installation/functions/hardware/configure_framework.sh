configure_framework() {
    local model
    cfg_framework_enable=false
    cfg_framework_model=""
    if [[ "${cfg_preset_loaded:-false}" == true ]]; then
        cfg_framework_enable="$(preset_bool frameworkEnable && echo true || echo false)"
        cfg_framework_model="$(preset_get frameworkModel)"
        return 0
    fi
    # The profile/form-factor choice already identifies the laptop family.
    # Desktops, ThinkPads, and generic laptops do not receive Framework
    # settings or an unrelated extra prompt.
    [[ "${CFG_FORM_FACTOR:-laptop}" == laptop ]] || return 0
    [[ "${CFG_LAPTOP_VENDOR:-generic}" == framework || "$cfg_profile" == framework* ]] || return 0

    local -a models=(13 16 12)
    if [[ "${INSTALLER_UI:-terminal}" == gtk ]]; then
        prompt_choice 'Framework model:' 13 models
        cfg_framework_model="$REPLY"
        cfg_framework_enable=true
        return 0
    fi
    printf '  1) Framework 13\n  2) Framework 16\n  3) Framework 12\n'
    while true; do
        read -r -p 'Framework model [1]: ' model
        model="${model:-1}"
        case "$model" in
            1) cfg_framework_model=13; break ;;
            2) cfg_framework_model=16; break ;;
            3) cfg_framework_model=12; break ;;
            *) printf 'Choose 1, 2, or 3.\n' >&2 ;;
        esac
    done
    cfg_framework_enable=true
}
