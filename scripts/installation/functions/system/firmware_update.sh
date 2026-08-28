firmware_update() {
    local answer
    if ! command -v fwupdmgr >/dev/null 2>&1; then
        say '[NOTE] fwupdmgr is unavailable; skipping firmware update check.'
        return 0
    fi
    read -r -p 'Check for and install available firmware updates now? [y/N] ' answer
    [[ "$answer" =~ ^([yY]|[yY][eE][sS])$ ]] || {
        say 'Firmware update check skipped.'
        return 0
    }
    command -v sudo >/dev/null 2>&1 || {
        say '[WARN] sudo is unavailable; cannot apply firmware updates.'
        return 0
    }
    say 'Refreshing firmware metadata...'
    if ! sudo fwupdmgr refresh --force; then
        say '[WARN] Firmware metadata refresh failed; continuing with the installer.'
        return 0
    fi
    say 'Applying available firmware updates...'
    local update_log
    update_log="$(mktemp)"
    if sudo fwupdmgr update 2>&1 | tee "$update_log"; then
        if grep -qiE 'no updatable|latest available firmware version|no available firmware' "$update_log" &&
           ! grep -qiE 'successfully (installed|updated)|firmware.*updated' "$update_log"; then
            say 'No firmware updates available; continuing.'
        else
            say 'Firmware update process completed. A reboot may be required.'
        fi
    else
        say '[WARN] Firmware update process failed or no compatible update was available.'
    fi
    rm -f -- "$update_log"
}
