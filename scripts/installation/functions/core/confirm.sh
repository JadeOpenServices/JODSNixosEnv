confirm() {
    local answer dialog_text
    if [[ "${INSTALLER_UI:-terminal}" == gtk ]]; then
        dialog_text=$'\n  '"$1"$'  \n'
        zenity --question --width=620 --title='GjallarOS installer' --text="$dialog_text" >/dev/null 2>&1
        return $?
    fi
    read -r -p "$1 [y/N] " answer
    [[ "$answer" =~ ^([yY]|[yY][eE][sS])$ ]]
}
