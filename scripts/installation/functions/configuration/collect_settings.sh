collect_settings() {
    local repo_root="$1" profiles_name="$2" shells_name="$3" editors_name="$4" browsers_name="$5" wms_name="$6" themes_name="$7"
    local -n profile_values="$profiles_name" shell_values="$shells_name" editor_values="$editors_name" browser_values="$browsers_name" wm_values="$wms_name" theme_values="$themes_name"
    local default_system default_profile default_shell default_theme default_username default_hostname
    local -a system_options default_editors default_browsers
    if [[ "${cfg_preset_loaded:-false}" == true ]]; then
        cfg_system="$(preset_get system)"; cfg_profile="$(preset_get profile)"
        cfg_hostname="$(preset_get hostname)"; cfg_username="$(preset_get username)"
        cfg_timezone="$(preset_get timezone)"; cfg_locale="$(preset_get locale)"
        cfg_name="$(preset_get name)"; cfg_email="$(preset_get email)"
        cfg_github_username="$(preset_get githubUsername)"
        cfg_dotfiles_dir="$(preset_get dotfilesDir)"; cfg_shell="$(preset_get shell)"
        [[ -n "$cfg_dotfiles_dir" ]] || cfg_dotfiles_dir="/home/$cfg_username/Documents/gjallarOS"
        cfg_dotfiles_dir="${cfg_dotfiles_dir//usernamehere/$cfg_username}"
        mapfile -t cfg_editors < <(preset_array editors)
        mapfile -t cfg_browsers < <(preset_array browsers)
        cfg_editors=("${cfg_editors[@]:-vscodium}"); cfg_browsers=("${cfg_browsers[@]:-librewolf}")
        cfg_preferred_editor="$(preset_get preferredEditor)"; cfg_preferred_browser="$(preset_get preferredBrowser)"
        cfg_wms=(hyprland); cfg_theme="$(preset_get theme)"
        cfg_background_normal="$(preset_get backgroundNormal)"
        cfg_background_work="$(preset_get backgroundWork)"
        cfg_background_gaming="$(preset_get backgroundGaming)"
        resolve_backgrounds "$repo_root" "$cfg_dotfiles_dir"
        cfg_enable_scrobbling=false; cfg_enable_lastfm=false; cfg_enable_listenbrainz=false
        cfg_lastfm_username=''; cfg_listenbrainz_username=''
        cfg_framework_enable=false; cfg_framework_model=''; cfg_luks_tpm2_enable=false
        cfg_docker_enable=false; cfg_ai_enable=false; cfg_ai_model='qwen2.5-coder:7b'
        cfg_ai_context_tokens=8192; cfg_ai_vram_mb=0
        configure_work_user; return 0
    fi
    system_options=("x86_64-linux" "aarch64-linux")
    default_system="$(uname -m)-linux"
    [[ "$default_system" == "x86_64-linux" || "$default_system" == "aarch64-linux" ]] || default_system="x86_64-linux"
    default_profile="${profile_values[0]}"; default_shell="${shell_values[0]}"; default_theme="${theme_values[0]}"
    default_username="${SUDO_USER:-${USER:-user}}"
    default_hostname="$(hostname 2>/dev/null || true)"
    # Fresh NixOS installs commonly still identify themselves as "nixos";
    # GjallarOS should offer its own stable hostname instead.
    [[ -z "$default_hostname" || "$default_hostname" == nixos ]] &&
        default_hostname="gjallarOS"
    default_editors=(vscodium); default_browsers=("${browser_values[0]}")

    prompt_choice 'System architecture:' "$default_system" system_options; cfg_system="$REPLY"
    prompt_choice 'Profile:' "$default_profile" "$profiles_name"; cfg_profile="$REPLY"
    prompt_value 'Hostname:' "$default_hostname"; cfg_hostname="$REPLY"
    prompt_value 'Username:' "$default_username"; cfg_username="$REPLY"
    configure_work_user
    prompt_value 'Timezone (IANA name):' 'Europe/Berlin'; cfg_timezone="$REPLY"
    prompt_value 'Locale:' 'en_US.UTF-8'; cfg_locale="$REPLY"
    prompt_value 'Full name (used by Git):' "$cfg_username"; cfg_name="$REPLY"
    prompt_value 'Email (used by Git):' "$cfg_username@example.com"; cfg_email="$REPLY"
    prompt_value 'GitHub username (optional, separate from Linux user):' ''; cfg_github_username="$REPLY"
    prompt_value 'Absolute dotfiles path:' "$repo_root"; cfg_dotfiles_dir="$REPLY"
    [[ "$cfg_dotfiles_dir" = /* ]] || die 'Dotfiles path must be absolute.'
    prompt_choice 'Login shell:' "$default_shell" "$shells_name"; cfg_shell="$REPLY"
    until prompt_multi 'Editors:' default_editors "$editors_name"; do :; done; cfg_editors=("${PROMPT_SELECTION[@]}")
    until prompt_multi 'Browsers:' default_browsers "$browsers_name"; do :; done; cfg_browsers=("${PROMPT_SELECTION[@]}")
    prompt_value 'Preferred editor command:' "${cfg_editors[0]}"; cfg_preferred_editor="$REPLY"
    prompt_value 'Preferred browser command:' "${cfg_browsers[0]}"; cfg_preferred_browser="$REPLY"
    cfg_wms=(hyprland)
    prompt_choice 'Theme:' "$default_theme" "$themes_name"; cfg_theme="$REPLY"
    cfg_background_normal=''
    cfg_background_work=''
    cfg_background_gaming=''
    cfg_enable_scrobbling=false
    cfg_enable_lastfm=false
    cfg_enable_listenbrainz=false
    cfg_lastfm_username=''
    cfg_listenbrainz_username=''
    cfg_framework_enable=false
    cfg_framework_model=''
    cfg_luks_tpm2_enable=false
    cfg_docker_enable=false
    cfg_ai_enable=false
    cfg_ai_model='qwen2.5-coder:7b'
    cfg_ai_context_tokens=8192
    cfg_ai_vram_mb=0
}
