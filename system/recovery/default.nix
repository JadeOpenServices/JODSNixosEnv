{
  config,
  lib,
  pkgs,
  settings,
  ...
}:
let
  tools = import ./tools.nix { inherit config pkgs; };
  inherit (tools) gjallarctl recovery recoveryExecutor;
in
lib.mkMerge [
  {
    assertions = lib.optional settings.jodsPrebootLockEnable {
      assertion =
        settings.endpointManagedDevice
        && settings.recoveryEnable
        && settings.secureBootEnable
        && settings.luksTpm2Enable;
      message = "JODS preboot locking requires explicit endpoint management enrollment, recoveryEnable, secureBootEnable, and luksTpm2Enable.";
    };
  }

  (lib.mkIf settings.recoveryEnable {
    boot.loader.timeout = lib.mkForce 5;

    specialisation.gjallar-recovery.configuration = {
      system.nixos.tags = [ "trusted-recovery" ];
      boot.kernelParams = [ "systemd.unit=multi-user.target" ];
      environment.systemPackages = [
        recovery
        recoveryExecutor
        gjallarctl
        pkgs.dosfstools
        pkgs.gptfdisk
        pkgs.parted
        pkgs.xorriso
      ];
      services.openssh.enable = lib.mkForce false;
      networking.firewall.enable = lib.mkForce true;

      environment.etc."gjallar/recovery".text = ''
        Trusted GjallarOS recovery entry.
        Sign in locally and run: sudo gjallar-recover audit
        Root disks require a verified human LUKS recovery credential.
      '';
    } // lib.optionalAttrs settings.endpointManagedDevice {
      services.jods-mdm-agent.recoveryExecutorEnable = true;
    };
  })

  (lib.mkIf (settings.recoveryEnable || settings.jodsPrebootLockEnable) {
    environment.etc."jods/preboot-policy".text = ''
      version=1
      recovery=${if settings.recoveryEnable then "signed-independent-xbootldr" else "disabled"}
      root_unlock=human-recovery-credential-only
      external_media_tpm_unlock=forbidden
      jods_lock=${if settings.jodsPrebootLockEnable then "measured-boot-required" else "disabled"}
      modes=repair,reinstall,fresh
      default_mode=repair
    '';
  })
]
