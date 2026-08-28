detect_ui() {
    local gtk_config_dir gtk_css
    # Keep the spacing scale in one place.  Dialog, content, row, and control
    # spacing are derived from these values so the layout stays proportional.
    local window_pad=14 outer_margin=12 outer_pad=6 content_margin=8 content_pad=8
    local widget_margin=4 widget_pad_y=8 widget_pad_x=12 row_margin_y=3 row_margin_x=6
    local button_margin_y=6 button_margin_x=5 button_pad_y=9 button_pad_x=16
    local list_pad=10 row_gap=8
    INSTALLER_UI=terminal
    if [[ "${cfg_preset_loaded:-false}" == true ]]; then
        printf '[INFO] User preset detected; using non-interactive terminal mode.\n'
        return 0
    fi
    if command -v zenity >/dev/null 2>&1 &&
       { [[ -n "${WAYLAND_DISPLAY:-}" ]] || [[ -n "${DISPLAY:-}" ]]; }; then
        # Use GTK's built-in engine for predictable Zenity rendering; the
        # stylesheet below supplies a consistent spacing scale.
        export GTK_THEME="Adwaita:dark"
        # GTK3 reads the user stylesheet, not /etc/gtk-3.0 reliably. Provide
        # the system stylesheet through a private config directory so Zenity
        # receives the spacing, palette, and button rules immediately.
        gtk_config_dir="${TMPDIR:-/tmp}/gjallar-installer-gtk.$$"
        mkdir -p "$gtk_config_dir/gtk-3.0" "$gtk_config_dir/gtk-4.0"
        gtk_css="$gtk_config_dir/gtk-3.0/gtk.css"
        # Use a per-run stylesheet: Adwaita supplies the base theme while these
        # rules add consistent spacing inside Zenity's nested option widgets.
        : > "$gtk_css"
        printf '%s\n' \
            "window { padding: ${window_pad}px; }" \
            'window > box, window > grid {' \
            "  margin: ${outer_margin}px; padding: ${outer_pad}px;" \
            '}' \
            'window > box > box, window > grid > box {' \
            "  margin: ${content_margin}px; padding: ${content_pad}px;" \
            '}' \
            'dialog .dialog-vbox, dialog .content-area, dialog .action-area {' \
            "  margin: ${content_margin}px; padding: ${content_pad}px;" \
            '}' \
            'dialog list, dialog treeview, dialog viewport, list, treeview, viewport {' \
            "  margin: ${content_margin}px; padding: ${list_pad}px;" \
            '}' \
            'treeview.view {' \
            "  padding: ${list_pad}px; border-spacing: ${row_gap}px;" \
            "  -GtkTreeView-vertical-separator: ${row_gap};" \
            "  -GtkTreeView-horizontal-separator: ${row_gap};" \
            '}' \
            'dialog list row, dialog treeview.view row, list row {' \
            "  margin: ${row_margin_y}px ${row_margin_x}px; padding: ${widget_pad_y}px ${widget_pad_x}px; min-height: ${widget_pad_y}px;" \
            '  border-radius: 8px;' \
            '}' \
            'dialog entry, dialog checkbutton, entry, checkbutton {' \
            "  margin: ${widget_margin}px ${row_margin_x}px; padding: ${widget_pad_y}px ${widget_pad_x}px;" \
            '}' \
            'window actionbar, window > box > actionbar {' \
            "  margin-top: ${content_margin}px; padding-top: ${outer_pad}px;" \
            '}' \
            'window button {' \
            "  margin: ${button_margin_y}px ${button_margin_x}px ${row_margin_y}px; padding: ${button_pad_y}px ${button_pad_x}px;" \
            '}' \
            'button {' \
            "  margin: ${button_margin_y}px ${button_margin_x}px ${row_margin_y}px;" \
            '}' > "$gtk_css"
        cp -- "$gtk_css" "$gtk_config_dir/gtk-4.0/gtk.css"
        export XDG_CONFIG_HOME="$gtk_config_dir"
        INSTALLER_UI=gtk
        printf '[INFO] GTK installer dialogs enabled.\n'
    else
        printf '[INFO] GTK dialogs unavailable (preset, Zenity, or display missing); using terminal UI.\n'
    fi
}
