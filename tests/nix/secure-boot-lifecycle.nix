{
  nixpkgs,
  system,
  lifecycleModule,
}:
let
  pkgs = nixpkgs.legacyPackages.${system};
  stub = pkgs.emptyDirectory;
  evaluated = nixpkgs.lib.nixosSystem {
    inherit system;
    modules = [
      lifecycleModule
      {
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
        luksTpm2Enable = false;
      };
    };
  };
  post = evaluated.config.systemd.services.installer-post-secure-boot;
in
# Started in parallel with enrollment, the continuation told the user to
# remove PK while firmware already was in Setup Mode (e2e-target, 2026-09-29).
assert builtins.elem "secure-boot-enrollment.service" post.after;
pkgs.runCommand "gjallar-secure-boot-lifecycle-check" { } "touch $out"
