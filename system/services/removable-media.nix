{
  lib,
  pkgs,
  settings,
  ...
}:
let
  usbguard = settings.usbguardEnable or false;

  seedRules = ../../generated/usbguard-rules.conf;
  runtimeRules = "/var/lib/usbguard/rules.conf";

  usbtrustd = pkgs.callPackage ../../pkgs/gjallar-usbtrustd { };

  usbtrustStart = pkgs.writeShellScript "gjallar-usbtrustd-start" ''
    ownerUID="$(${pkgs.coreutils}/bin/id -u ${lib.escapeShellArg settings.username})"

    exec ${usbtrustd}/bin/gjallar-usbtrustd \
      --socket /run/gjallar-usbtrust/control.sock \
      --owner-uid "$ownerUID" \
      --oddc-root ${usbtrustd}/share/gjallarOS/oddc \
      --oddc-model ${lib.escapeShellArg (settings.oddcModel or "")} \
      --state-dir /var/lib/gjallar-usbtrust \
      --usbguard-binary ${pkgs.usbguard}/bin/usbguard
  '';
in
{
  # The desktop user may reach the read-only broker socket.
  # Authoritative trust state itself remains root-only.
  users.groups.gjallar-usbtrust = lib.mkIf usbguard { };

  users.users.${settings.username}.extraGroups = lib.mkIf usbguard [ "gjallar-usbtrust" ];

  # Provides privileged storage operations over D-Bus.
  # Nothing is mounted automatically.
  services.udisks2.enable = true;

  assertions = lib.optionals usbguard [
    {
      assertion = builtins.pathExists seedRules;
      message = "USBGuard is enabled but generated/usbguard-rules.conf is missing.";
    }
    {
      assertion = (settings.oddcModel or "") != "";
      message = "USBGuard requires an already-selected ODDC model.";
    }
  ];

  services.usbguard = lib.mkIf usbguard {
    enable = true;

    # Mutable machine-local policy. Unlike services.usbguard.rules,
    # this allows explicitly approved permanent devices to persist.
    rules = null;
    ruleFile = runtimeRules;

    implicitPolicyTarget = "block";
    presentDevicePolicy = "apply-policy";
    presentControllerPolicy = "keep";
    insertedDevicePolicy = "apply-policy";
    restoreControllerDeviceState = true;

    # Parent hashes remain available, without permanently binding
    # portable external devices to one USB port.
    deviceRulesWithPort = false;
  };

  systemd.services.gjallar-usbtrustd = lib.mkIf usbguard {
    description = "GjallarOS USB trust broker";

    wantedBy = [ "multi-user.target" ];
    requires = [ "usbguard.service" ];
    after = [ "usbguard.service" ];

    path = [
      pkgs.tpm2-tools
    ];

    serviceConfig = {
      Type = "simple";

      User = "root";
      Group = "gjallar-usbtrust";

      ExecStart = usbtrustStart;

      RuntimeDirectory = "gjallar-usbtrust";
      RuntimeDirectoryMode = "0750";

      StateDirectory = "gjallar-usbtrust";
      StateDirectoryMode = "0700";

      UMask = "0077";

      Restart = "on-failure";
      RestartSec = "2s";

      NoNewPrivileges = true;
      PrivateTmp = true;
      ProtectSystem = "strict";
      ProtectHome = true;
      ProtectKernelTunables = true;
      ProtectKernelModules = true;
      ProtectControlGroups = true;
      RestrictSUIDSGID = true;
      LockPersonality = true;
    };
  };

  systemd.services.usbguard.preStart = lib.mkIf usbguard (
    lib.mkBefore ''
          install -d -m 0700 \
            /var/lib/usbguard \
            /var/lib/usbguard/IPCAccessControl.d

          # Seed the persistent policy exactly once from the validated
          # internal-device policy. Never overwrite later user trust.
          if [ ! -s ${runtimeRules} ]; then
            install -m 0600 ${seedRules} ${runtimeRules}
          fi

          cat > /var/lib/usbguard/IPCAccessControl.d/${settings.username} <<'ACL'
      Devices=list,listen
      Policy=list
      Exceptions=listen
      ACL

          chmod 0600 \
            /var/lib/usbguard/IPCAccessControl.d/${settings.username} \
            ${runtimeRules}
    ''
  );
}
