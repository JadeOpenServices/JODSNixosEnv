prompt_choice() {
    local prompt="$1" default="$2" options_name="$3" selection index dialog_text
    local -n options_ref="$options_name"
    if [[ "${INSTALLER_UI:-terminal}" == gtk ]]; then
        local -a rows=()
        for index in "${!options_ref[@]}"; do rows+=("${options_ref[index]}"); done
        dialog_text=$'\n  '"$prompt"$'  \n'
        while ! REPLY="$(zenity --list --hide-header --print-column=1 --width=680 --height=460 --title='GjallarOS installer' --text="$dialog_text" --column='Option' "${rows[@]}" 2>/dev/null)"; do
            gtk_cancel_prompt || continue
        done
        REPLY="${REPLY:-$default}"
        return 0
    fi
    printf '\n%s\n' "$prompt"
    for index in "${!options_ref[@]}"; do
        printf '  %d) %s%s\n' "$((index + 1))" "${options_ref[index]}" \
            "$([[ "${options_ref[index]}" == "$default" ]] && printf ' (default)')"
    done
    while true; do
        read -r -p "Selection [$default]: " selection
        [[ -z "$selection" ]] && { REPLY="$default"; return; }
        if [[ "$selection" =~ ^[0-9]+$ ]] && (( selection >= 1 && selection <= ${#options_ref[@]} )); then
            REPLY="${options_ref[selection - 1]}"
            return
        fi
        printf 'Choose a number from 1 to %d.\n' "${#options_ref[@]}" >&2
    done
}
