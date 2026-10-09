{ noctalia }:

# Keep the pinned upstream package and dependencies; only repair the shell's
# metadata admission and generic tray behavior, and let notification cards be
# swiped away.
noctalia.overrideAttrs (old: {
  patches = (old.patches or [ ]) ++ [
    ./media-tray.patch
    ./notification-swipe.patch
  ];
})
