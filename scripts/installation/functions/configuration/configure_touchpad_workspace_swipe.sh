configure_touchpad_workspace_swipe() {
    local answer requested
    cfg_touchpad_workspace_swipe=true

    if [[ "${cfg_preset_loaded:-false}" == true ]]; then
        requested="$(preset_get touchpadWorkspaceSwipe)"
        [[ "$requested" == false ]] && cfg_touchpad_workspace_swipe=false
        return 0
    fi

    if [[ "${INSTALLER_UI:-terminal}" == gtk ]]; then
        confirm 'Enable three-finger touchpad swipes to switch workspaces?' && answer=y || answer=n
    else
        read -r -p 'Enable three-finger touchpad swipes to switch workspaces? [Y/n] ' answer
    fi
    [[ "$answer" =~ ^([nN]|[nN][oO])$ ]] && cfg_touchpad_workspace_swipe=false
}
