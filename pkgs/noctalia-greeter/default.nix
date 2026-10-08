{ noctalia-greeter }:

# Keep the pinned upstream package; the root apply-appearance helper reads a
# root-owned snapshot of the staging folder so staged symlinks cannot leak
# files into the greeter state.
noctalia-greeter.overrideAttrs (old: {
  patches = (old.patches or [ ]) ++ [ ./staging-snapshot.patch ];
})
