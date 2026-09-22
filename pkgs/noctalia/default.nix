{ noctalia }:

# Keep the pinned upstream package and dependencies; only repair the shell's
# metadata admission and generic tray behavior.
noctalia.overrideAttrs (old: {
  patches = (old.patches or [ ]) ++ [ ./media-tray.patch ];
})
