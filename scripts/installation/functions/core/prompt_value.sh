prompt_value() {
    local prompt="$1" default="$2" value dialog_text
    if [[ "${INSTALLER_UI:-terminal}" == gtk ]]; then
        dialog_text=$'\n  '"$prompt"$'  \n'
        while ! value="$(zenity --entry --width=620 --title='GjallarOS installer' --text="$dialog_text" --entry-text="$default" 2>/dev/null)"; do
            gtk_cancel_prompt || continue
        done
        REPLY="${value:-$default}"
        return 0
    fi
    read -r -p "$prompt [$default]: " value
    REPLY="${value:-$default}"
}
