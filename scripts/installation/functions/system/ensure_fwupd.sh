ensure_fwupd() {
    if command -v fwupdmgr >/dev/null 2>&1 &&
       command -v systemctl >/dev/null 2>&1 &&
       command -v sudo >/dev/null 2>&1; then
        if sudo systemctl start fwupd.service 2>/dev/null; then
            # fwupd is often socket-activated/static; enabling may correctly
            # return non-zero, so do not treat that as a startup failure.
            sudo systemctl enable fwupd.service 2>/dev/null || true
        else
            say 'fwupd is installed but its service is not available in the active system yet.'
            say 'It will become available after the GjallarOS rebuild.'
        fi
    fi
}
