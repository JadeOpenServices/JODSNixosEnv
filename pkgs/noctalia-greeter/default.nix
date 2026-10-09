{ noctalia-greeter }:

# Keep the pinned upstream package; the root apply-appearance helper reads a
# root-owned snapshot of the staging folder so staged symlinks cannot leak
# files into the greeter state. menu-click.patch keeps a click on a control
# alive when the press moves focus and the menus get rebuilt.
# wlr-log-level.patch stops wlroots debug lines (one per frame with a
# software cursor) from flooding the journal.
noctalia-greeter.overrideAttrs (old: {
  patches = (old.patches or [ ]) ++ [
    ./staging-snapshot.patch
    ./menu-click.patch
    ./wlr-log-level.patch
  ];
})
