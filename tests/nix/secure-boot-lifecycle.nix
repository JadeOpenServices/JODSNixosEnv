{
  nixpkgs,
  system,
  lifecycleModule,
  measuredBootModule,
}:
let
  pkgs = nixpkgs.legacyPackages.${system};
  stub = pkgs.emptyDirectory;
  evaluated = nixpkgs.lib.nixosSystem {
    inherit system;
    modules = [
      lifecycleModule
      measuredBootModule
      {
        boot.initrd.luks.devices.cryptroot.device =
          "/dev/disk/by-uuid/00000000-0000-0000-0000-000000000000";
        system.stateVersion = "26.05";
        boot.loader.grub.enable = false;
        fileSystems."/" = {
          device = "/dev/null";
          fsType = "ext4";
        };
      }
    ];
    specialArgs = {
      gjallarctlPackage = stub;
      gjallarSecureBootArtifactVerifier = stub;
      gjallarSecureBootOwnershipVerifier = stub;
      settings = {
        secureBootEnable = true;
        luksTpm2Enable = true;
      };
    };
  };
  post = evaluated.config.systemd.services.installer-post-secure-boot;
  enrollment = evaluated.config.systemd.services.secure-boot-enrollment;
  tpm2 = evaluated.config.systemd.services.measured-boot-tpm2-enrollment;
in
# The TPM2 unlock self-test needs systemd's cryptsetup token plugin, which
# cryptsetup does not find by default (e2e-target, 2026-09-29).
assert nixpkgs.lib.hasInfix
  (builtins.unsafeDiscardStringContext "--external-tokens-path=${evaluated.config.systemd.package}/lib/cryptsetup")
  (builtins.unsafeDiscardStringContext tpm2.script);
# The TPM2 passphrase prompt needs Plymouth, which greetd quits (e2e-target,
# 2026-09-29: prompt hidden behind the greeter, enrollment hung).
assert builtins.elem "display-manager.service" tpm2.before;
assert builtins.elem "greetd.service" tpm2.before;
# gjallarctl runs sbctl verify in-process after enrolling; without lsblk the
# enrollment transaction failed after the keys were written (e2e-target,
# 2026-09-29).
assert builtins.elem pkgs.util-linux enrollment.path;
# Started in parallel with enrollment, the continuation told the user to
# remove PK while firmware already was in Setup Mode (e2e-target, 2026-09-29).
assert builtins.elem "secure-boot-enrollment.service" post.after;
pkgs.runCommand "gjallar-secure-boot-lifecycle-check" { } "touch $out"
