configure_auto_reboot() {
    local repo_root="$1" answer
    cfg_auto_reboot=false
    [[ -f "$repo_root/user.config.json" ]] || return 0
    [[ "${cfg_preset_loaded:-false}" == true ]] && { cfg_auto_reboot="$(preset_bool autoReboot && echo true || echo false)"; return 0; }
    say "Found local user.config.json; it is machine-local and will not be committed."
    if [[ "${INSTALLER_UI:-terminal}" == gtk ]]; then
        confirm 'Automatically reboot after a successful rebuild?' && answer=y || answer=n
    else
        read -r -p 'Automatically reboot after a successful rebuild? [y/N] ' answer
    fi
    [[ "$answer" =~ ^([yY]|[yY][eE][sS])$ ]] && cfg_auto_reboot=true
}
