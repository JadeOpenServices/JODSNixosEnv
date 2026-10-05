{
  nixpkgs,
  system,
  bootModule,
  hardeningModule,
}:
let
  evaluated = nixpkgs.lib.nixosSystem {
    inherit system;
    modules = [
      bootModule
      hardeningModule
      {
        system.stateVersion = "26.05";
        fileSystems."/".device = "/dev/null";
      }
    ];
    specialArgs.settings.debugFunctions = false;
  };
  config = evaluated.config;
  sysctl = config.boot.kernel.sysctl;
in
assert config.boot.loader.systemd-boot.editor == false;
# Leftover Secure Boot UKIs panic once their closures are collected.
assert nixpkgs.lib.hasInfix "/boot/EFI/Linux/nixos-generation-*.efi"
  config.boot.loader.systemd-boot.extraInstallCommands;
assert sysctl."net.ipv4.conf.all.accept_redirects" == 0;
assert sysctl."net.ipv6.conf.all.accept_redirects" == 0;
assert sysctl."net.ipv4.conf.all.send_redirects" == 0;
assert builtins.elem "d /var/lib/gjallarOS/passwords 0700 root root -" config.systemd.tmpfiles.rules;
nixpkgs.legacyPackages.${system}.runCommand "gjallar-security-baseline-check" { } ''
  touch "$out"
''
