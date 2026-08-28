configure_ai() {
    local answer
    cfg_ai_enable=false
    [[ "${cfg_preset_loaded:-false}" == true ]] && { cfg_ai_enable="$(preset_bool aiEnable && echo true || echo false)"; return 0; }
    if [[ "${INSTALLER_UI:-terminal}" == gtk ]]; then
        confirm 'Enable local AI tools (Ollama, OpenCode and research helper)?' || return 0
        answer=y
    else
        read -r -p 'Enable local AI tools (Ollama, OpenCode and research helper)? [y/N] ' answer
    fi
    [[ "$answer" =~ ^([yY]|[yY][eE][sS])$ ]] && cfg_ai_enable=true
}
