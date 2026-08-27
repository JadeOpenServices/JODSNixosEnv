prompt_choice() {
    local prompt="$1" default="$2" options_name="$3" selection index
    local -n options_ref="$options_name"
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
