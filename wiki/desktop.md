# Desktop

Hyprland is the only supported window manager. Home Manager generates
`~/.config/hypr/hyprland.conf`.

- Keybinds: `user/wm/hyprland/keybinds.json`.
- Noctalia shell: `user/wm/shells/noctalia/`.
- Themes: `themes/` and `themes/lib/`.
- Login screen: Noctalia Greeter.

GTK installer dialogs use Zenity when available and otherwise fall back to the
terminal. Cancelled GTK prompts require explicit quit confirmation.
