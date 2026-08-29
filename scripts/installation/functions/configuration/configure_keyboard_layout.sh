configure_keyboard_layout() {
    local requested
    if [[ "${cfg_preset_loaded:-false}" == true ]]; then
        requested="$(preset_get keyboardLayout)"
    else
        prompt_value 'Keyboard layout (for example: de or us):' 'de'
        requested="$REPLY"
    fi

    requested="${requested,,}"
    [[ -n "$requested" ]] || requested='de'
    [[ "$requested" =~ ^[a-z0-9,_+-]+$ ]] ||
        die 'Keyboard layout may contain only letters, numbers, commas, plus, underscore, and hyphen.'

    # The default XKB German map is the standard Latin-1-capable layout.
    # `de(latin1)` is not a valid variant in current xkeyboard-config.
    if [[ "$requested" == 'de' || "$requested" == 'de-latin1' ]]; then
        cfg_keyboard_layout='de'
        cfg_keyboard_variant=''
    else
        cfg_keyboard_layout="$requested"
        cfg_keyboard_variant=''
    fi
}
