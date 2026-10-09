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
        # As nixos-generate-config writes it.
        fileSystems."/boot" = {
          device = "/dev/null";
          fsType = "vfat";
          options = [
            "fmask=0022"
            "dmask=0022"
          ];
        };
        services.greetd.enable = true;
        services.greetd.settings.default_session.command = "true";
      }
    ];
    specialArgs.settings.debugFunctions = false;
  };
  config = evaluated.config;
  sysctl = config.boot.kernel.sysctl;
  pam = config.security.pam.services;
  # The password (change) stack keeps nullok upstream; only auth matters.
  authAllowsNull =
    service:
    builtins.any (line: nixpkgs.lib.hasPrefix "auth " line && nixpkgs.lib.hasInfix "nullok" line) (
      nixpkgs.lib.splitString "\n" pam.${service}.text
    );
in
assert config.boot.loader.systemd-boot.editor == false;
# Leftover Secure Boot UKIs panic once their closures are collected.
assert nixpkgs.lib.hasInfix "/boot/EFI/Linux/nixos-generation-*.efi"
  config.boot.loader.systemd-boot.extraInstallCommands;
# The ESP holds the boot loader random seed; the last mask wins.
assert nixpkgs.lib.lists.last config.fileSystems."/boot".options == "dmask=0077";
assert builtins.elem "fmask=0077" (nixpkgs.lib.drop 2 config.fileSystems."/boot".options);
assert sysctl."net.ipv4.conf.all.accept_redirects" == 0;
assert sysctl."net.ipv6.conf.all.accept_redirects" == 0;
assert sysctl."net.ipv4.conf.all.send_redirects" == 0;
assert builtins.elem "d /var/lib/gjallarOS/passwords 0700 root root -"
  config.systemd.tmpfiles.rules;
# Generic pkexec stays refused for non-root callers.
assert nixpkgs.lib.hasInfix
  ''action.id == "org.freedesktop.policykit.exec" && subject.user != "root"''
  config.security.polkit.extraConfig;
assert !(authAllowsNull "greetd");
assert !(authAllowsNull "login");
nixpkgs.legacyPackages.${system}.runCommand "gjallar-security-baseline-check" { } ''
  touch "$out"
''
