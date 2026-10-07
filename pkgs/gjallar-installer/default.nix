# The disk-provisioning binaries. They can repartition and reinstall a
# machine, so the installed system never carries them on its PATH: only the
# recovery specialisation, the recovery-maintenance initrd and the recovery
# ISO include this package.
{ callPackage }:
let
  gjallarctl = callPackage ../gjallarctl { };
in
gjallarctl.overrideAttrs (old: {
  pname = "gjallar-installer";

  subPackages = [
    "cmd/gjallar-installer"
    "cmd/gjallar-recovery-maintenance"
  ];

  # The installer runs gjallarctl, which no longer sits next to it.
  ldflags = old.ldflags ++ [
    "-X github.com/bakanura/gjallarOS/internal/installer/control.Path=${gjallarctl}/bin/gjallarctl"
  ];

  meta = old.meta // {
    description = "GjallarOS installer and recovery-storage maintenance";
    mainProgram = "gjallar-installer";
  };
})
