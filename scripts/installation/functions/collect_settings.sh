collect_settings() {
    local repo_root="$1" profiles_name="$2" shells_name="$3" editors_name="$4" browsers_name="$5" wms_name="$6" themes_name="$7"
    local -n profile_values="$profiles_name" shell_values="$shells_name" editor_values="$editors_name" browser_values="$browsers_name" wm_values="$wms_name" theme_values="$themes_name"
    local default_system default_profile default_shell default_theme default_username default_hostname
    local -a system_options default_editors default_browsers default_wms
    system_options=("x86_64-linux" "aarch64-linux")
    default_system="$(uname -m)-linux"
    [[ "$default_system" == "x86_64-linux" || "$default_system" == "aarch64-linux" ]] || default_system="x86_64-linux"
    default_profile="${profile_values[0]}"; default_shell="${shell_values[0]}"; default_theme="${theme_values[0]}"
    default_username="${SUDO_USER:-${USER:-user}}"; default_hostname="$(hostname 2>/dev/null || printf alfheim)"
    default_editors=("${editor_values[0]}"); default_browsers=("${browser_values[0]}"); default_wms=("${wm_values[0]}")

    prompt_choice 'System architecture:' "$default_system" system_options; cfg_system="$REPLY"
    prompt_choice 'Profile:' "$default_profile" "$profiles_name"; cfg_profile="$REPLY"
    prompt_value 'Hostname:' "$default_hostname"; cfg_hostname="$REPLY"
    prompt_value 'Username:' "$default_username"; cfg_username="$REPLY"
    configure_work_user
    prompt_value 'Timezone (IANA name):' 'Europe/Berlin'; cfg_timezone="$REPLY"
    prompt_value 'Locale:' 'en_US.UTF-8'; cfg_locale="$REPLY"
    prompt_value 'Full name (used by Git):' "$cfg_username"; cfg_name="$REPLY"
    prompt_value 'Email (used by Git):' "$cfg_username@example.com"; cfg_email="$REPLY"
    prompt_value 'Absolute dotfiles path:' "$repo_root"; cfg_dotfiles_dir="$REPLY"
    [[ "$cfg_dotfiles_dir" = /* ]] || die 'Dotfiles path must be absolute.'
    prompt_choice 'Login shell:' "$default_shell" "$shells_name"; cfg_shell="$REPLY"
    until prompt_multi 'Editors:' default_editors "$editors_name"; do :; done; cfg_editors=("${PROMPT_SELECTION[@]}")
    until prompt_multi 'Browsers:' default_browsers "$browsers_name"; do :; done; cfg_browsers=("${PROMPT_SELECTION[@]}")
    prompt_value 'Preferred editor command:' "${cfg_editors[0]}"; cfg_preferred_editor="$REPLY"
    prompt_value 'Preferred browser command:' "${cfg_browsers[0]}"; cfg_preferred_browser="$REPLY"
    until prompt_multi 'Window managers:' default_wms "$wms_name"; do :; done; cfg_wms=("${PROMPT_SELECTION[@]}")
    prompt_choice 'Theme:' "$default_theme" "$themes_name"; cfg_theme="$REPLY"
    cfg_enable_scrobbling=false
    cfg_enable_lastfm=false
    cfg_enable_listenbrainz=false
    cfg_lastfm_username=''
    cfg_listenbrainz_username=''
    cfg_framework_enable=false
    cfg_framework_model=''
    cfg_luks_tpm2_enable=false
}
