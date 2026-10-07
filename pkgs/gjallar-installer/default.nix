# The disk-provisioning binaries. They can repartition and reinstall a
# machine, so the installed system never carries them on its PATH: only the
# recovery specialisation, the recovery-maintenance initrd and the recovery
# ISO include this package.
{ callPackage }:
(callPackage ../gjallarctl { }).overrideAttrs (old: {
  pname = "gjallar-installer";

  subPackages = [
    "cmd/gjallar-installer"
    "cmd/gjallar-recovery-maintenance"
  ];

  meta = old.meta // {
    description = "GjallarOS installer and recovery-storage maintenance";
    mainProgram = "gjallar-installer";
  };
})
