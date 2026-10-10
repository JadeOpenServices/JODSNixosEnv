{ noctalia }:

# Keep the pinned upstream package and dependencies; only repair the shell's
# metadata admission and generic tray behavior, let notification cards be
# swiped away, show Bluetooth devices by their advertised name with nameless
# ones hidden by default, and treat a touch long press as a right click.
noctalia.overrideAttrs (old: {
  patches = (old.patches or [ ]) ++ [
    ./media-tray.patch
    ./notification-swipe.patch
    ./bluetooth.patch
    ./touch-long-press.patch
  ];
})
