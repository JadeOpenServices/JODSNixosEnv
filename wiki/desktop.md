# Desktop

Hyprland is the only supported window manager. Home Manager generates
`~/.config/hypr/hyprland.conf`.

- Keybinds: `user/wm/hyprland/keybinds.json`.
- AGS shell: `user/wm/shells/ags/`.
- Themes: `themes/` and `themes/lib/`.
- Login screen: SDDM with `catppuccin-mocha-mauve`.

GTK installer dialogs use Zenity when available and otherwise fall back to the
terminal. Cancelled GTK prompts require explicit quit confirmation.
