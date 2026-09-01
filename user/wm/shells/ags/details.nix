{...}:
rec {
    # Keep AGS installed for its optional tools, but do not make a graphical
    # session depend on its JavaScript runtime starting correctly.
    launchCommands = [ ];
    binds = {
        launcher = "fuzzel";
        volumeUp = "pulsemixer --change-volume +2";
        volumeDown = "pulsemixer --change-volume -2";
        brightnessUp = "brightnessctl s +5%";
        brightnessDown = "brightnessctl s 5%-";
        lock = "hyprlock";
        screenshot = "wl-copy < $(grimshot --notify save area $XDG_PICTURES_DIR/Screenshots/$(TZ=utc date +'screenshot_%Y-%m-%d-%H%M%S.%3N.png'))";
        micMute = "pulsemixer --id $(pulsemixer --list-sources | cut -f3 | grep 'Default' | cut -d ',' -f 1 | cut -c 6-) --toggle-mute";
        mediaNext = "hyprmusic next";
        mediaPrev = "hyprmusic previous";
        mediaToggle = "hyprmusic play-pause";
    };
}
