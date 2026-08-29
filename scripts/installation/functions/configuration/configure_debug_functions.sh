configure_debug_functions() {
    cfg_debug_functions=false
    [[ "${cfg_preset_loaded:-false}" == true ]] || return 0

    # `debugFunctions` is the documented preset key. Accept the requested
    # hyphenated spelling too so existing local presets remain forgiving.
    if preset_bool debugFunctions || preset_bool 'Debug-Functions'; then
        cfg_debug_functions=true
    fi
}
