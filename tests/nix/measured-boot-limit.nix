{
  nixpkgs,
  system,
  lanzabooteModule,
  secureBootModule,
}:
let
  evaluate =
    luksTpm2Enable: recoveryEnable:
    (nixpkgs.lib.nixosSystem {
      inherit system;
      modules = [
        lanzabooteModule
        secureBootModule
        {
          system.stateVersion = "26.05";
          fileSystems."/".device = "/dev/null";
          # Same shape as system/recovery: trusted recovery + maintenance.
          specialisation.gjallar-recovery.configuration = { };
          specialisation.gjallar-recovery-maintenance.configuration = { };
        }
      ];
      specialArgs.settings = {
        secureBootEnable = true;
        inherit luksTpm2Enable recoveryEnable;
      };
    }).config;
  measured = (evaluate true false).boot.lanzaboote;
  unmeasured = (evaluate false false).boot.lanzaboote;
  recovery = evaluate true true;
in
# systemd-pcrlock: at most 8 alternatives per PCR; PCR 4 sees every UKI.
assert measured.measuredBoot.enable;
assert measured.configurationLimit * 3 <= 8;
assert measured.configurationLimit == 2;
assert unmeasured.configurationLimit == 8;
# With recovery, older generations are picked there, not in the boot menu.
assert recovery.boot.lanzaboote.configurationLimit == 1;
# The wrapped hook also drops systemd-boot's stale entries.
assert nixpkgs.lib.hasInfix "install-lanzaboote" "${recovery.system.build.installBootLoader}";
nixpkgs.legacyPackages.${system}.runCommand "gjallar-measured-boot-limit-check" { } ''
  touch "$out"
''
