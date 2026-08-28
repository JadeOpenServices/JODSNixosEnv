configure_framework() {
    local answer model
    cfg_framework_enable=false
    cfg_framework_model=""
    if [[ "${cfg_preset_loaded:-false}" == true ]]; then
        cfg_framework_enable="$(preset_bool frameworkEnable && echo true || echo false)"
        cfg_framework_model="$(preset_get frameworkModel)"
        return 0
    fi
    read -r -p 'Is this a Framework laptop? [y/N] ' answer
    [[ "$answer" =~ ^([yY]|[yY][eE][sS])$ ]] || return 0
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
