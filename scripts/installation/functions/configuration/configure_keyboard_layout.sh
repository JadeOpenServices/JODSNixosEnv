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

    # XKB calls the common German Latin-1 layout `de` with variant `latin1`.
    # Accept the natural installer input while generating valid XKB settings.
    if [[ "$requested" == 'de' || "$requested" == 'de-latin1' ]]; then
        cfg_keyboard_layout='de'
        cfg_keyboard_variant='latin1'
    else
        cfg_keyboard_layout="$requested"
        cfg_keyboard_variant=''
    fi
}
