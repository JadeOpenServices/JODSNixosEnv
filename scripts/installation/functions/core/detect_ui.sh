detect_ui() {
    local gtk_config_dir gtk_css
    INSTALLER_UI=terminal
    if [[ "${cfg_preset_loaded:-false}" == true ]]; then
        printf '[INFO] User preset detected; using non-interactive terminal mode.\n'
        return 0
    fi
    if command -v zenity >/dev/null 2>&1 &&
       { [[ -n "${WAYLAND_DISPLAY:-}" ]] || [[ -n "${DISPLAY:-}" ]]; }; then
        # Use GTK's built-in engine for predictable Zenity rendering; the
        # appended stylesheet below supplies the Catppuccin palette.
        export GTK_THEME="Adwaita:dark"
        # GTK3 reads the user stylesheet, not /etc/gtk-3.0 reliably. Provide
        # the system stylesheet through a private config directory so Zenity
        # receives the spacing, palette, and button rules immediately.
        gtk_config_dir="${TMPDIR:-/tmp}/gjallar-installer-gtk.$$"
        mkdir -p "$gtk_config_dir/gtk-3.0" "$gtk_config_dir/gtk-4.0"
        gtk_css="$gtk_config_dir/gtk-3.0/gtk.css"
        # Use an empty per-run stylesheet: Adwaita supplies its own stable
        # spacing and avoids inherited theme overrides in Zenity.
        : > "$gtk_css"
        printf '%s\n' \
            'window { padding: 12px; }' \
            'window > box, window > grid {' \
            '  margin: 10px; padding: 4px;' \
            '}' \
            'window > box > box, window > grid > box {' \
            '  margin: 3px; padding: 2px;' \
            '}' \
            'window actionbar, window > box > actionbar {' \
            '  margin-top: 12px; padding-top: 6px;' \
            '}' \
            'window button {' \
            '  margin: 6px 5px 3px 5px; padding: 7px 14px;' \
            '}' \
            'button {' \
            '  margin: 7px 5px 3px 5px;' \
            '}' > "$gtk_css"
        cp -- "$gtk_css" "$gtk_config_dir/gtk-4.0/gtk.css"
        export XDG_CONFIG_HOME="$gtk_config_dir"
        INSTALLER_UI=gtk
        printf '[INFO] GTK installer dialogs enabled.\n'
    else
        printf '[INFO] GTK dialogs unavailable (preset, Zenity, or display missing); using terminal UI.\n'
    fi
}
