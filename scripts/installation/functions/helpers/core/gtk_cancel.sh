gtk_cancel_prompt() {
    while true; do
        if zenity --question --width=620 --title='GjallarOS installer' \
            --text='  Cancel this prompt and quit the installer?  ' \
            --ok-label='Yes, quit' --cancel-label='No, return to prompt' >/dev/null 2>&1; then
            say 'Installer cancelled by user; no further actions were performed.'
            exit 0
        fi
        return 1
    done
}
