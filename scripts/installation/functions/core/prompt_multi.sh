prompt_multi() {
    local prompt="$1" defaults_name="$2" options_name="$3" selection item index dialog_text
    local -n defaults_ref="$defaults_name" options_ref="$options_name"
    if [[ "${INSTALLER_UI:-terminal}" == gtk ]]; then
        local -a rows=() selected
        for index in "${!options_ref[@]}"; do
            if printf '%s\n' "${defaults_ref[@]}" | grep -Fxq "${options_ref[index]}"; then rows+=(TRUE); else rows+=(FALSE); fi
            rows+=("${options_ref[index]}")
        done
        dialog_text=$'\n  '"$prompt"$'  \n'
        while ! selected="$(zenity --list --hide-header --checklist --multiple --print-column=2 --separator='|' --width=720 --height=500 --title='GjallarOS installer' --text="$dialog_text" --column='Select' --column='Option' "${rows[@]}" 2>/dev/null)"; do
            gtk_cancel_prompt || continue
        done
        IFS='|' read -r -a PROMPT_SELECTION <<< "$selected"
        ((${#PROMPT_SELECTION[@]})) || PROMPT_SELECTION=("${defaults_ref[@]}")
        return 0
    fi
    printf '\n%s\n' "$prompt"
    for index in "${!options_ref[@]}"; do
        printf '  %d) %s\n' "$((index + 1))" "${options_ref[index]}"
    done
    read -r -p "Selections (space-separated; default: ${defaults_ref[*]}): " selection
    if [[ -z "$selection" ]]; then
        PROMPT_SELECTION=("${defaults_ref[@]}")
        return
    fi
    PROMPT_SELECTION=()
    for item in $selection; do
        [[ "$item" =~ ^[0-9]+$ ]] && (( item >= 1 && item <= ${#options_ref[@]} )) || {
            printf 'Choose space-separated numbers from 1 to %d.\n' "${#options_ref[@]}" >&2
            return 1
        }
        PROMPT_SELECTION+=("${options_ref[item - 1]}")
    done
}
