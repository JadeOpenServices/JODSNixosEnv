prompt_multi() {
    local prompt="$1" defaults_name="$2" options_name="$3" selection item index
    local -n defaults_ref="$defaults_name" options_ref="$options_name"
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
