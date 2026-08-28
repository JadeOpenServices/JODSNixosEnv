configure_work_user() {
    local answer suffix
    cfg_work_user_enable=false; cfg_work_username=""
    if [[ "${cfg_preset_loaded:-false}" == true ]]; then
        cfg_work_user_enable="$(preset_bool workUserEnable && echo true || echo false)"
        cfg_work_username="$(preset_get workUsername)"
        return 0
    fi
    read -r -p 'Create a separate work account? [y/N] ' answer
    [[ "$answer" =~ ^([yY]|[yY][eE][sS])$ ]] || return 0
    prompt_value 'Work account suffix:' '-corp'; suffix="$REPLY"
    [[ "$suffix" == -* ]] || suffix="-$suffix"
    cfg_work_user_enable=true; cfg_work_username="${cfg_username}${suffix}"
}
