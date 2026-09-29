{
  config,
  lib,
  pkgs,
  settings,
  ...
}:
let
  cfg = config.services.gjallar-usbtrust;
  configuredTPMHandle = settings.usbTrustTpmHandle or "";
  usbtrustd = pkgs.callPackage ../../pkgs/gjallar-usbtrustd { };
  resolved = pkgs.writeText "gjallar-usbtrust-oddc.json" (builtins.toJSON config.oddc.resolved);

  usbTrustAskpass = pkgs.writeShellScript "gjallar-usbtrust-askpass" ''
    exec ${pkgs.zenity}/bin/zenity \
      --password \
      --title="Authorize USB device" \
      --text="Fingerprint authentication was unavailable, failed, or timed out. Enter your password to authorize this USB decision."
  '';
  start = pkgs.writeShellScript "gjallar-usbtrustd-start" ''
    ownerUID="$(${pkgs.coreutils}/bin/id -u ${lib.escapeShellArg settings.username})"
    exec ${usbtrustd}/bin/gjallar-usbtrustd \
      --socket /run/gjallar-usbtrust/control.sock \
      --owner-uid "$ownerUID" \
      --oddc-resolved ${resolved} \
      --oddc-model ${
        lib.escapeShellArg (if config.oddc.device == null then "" else config.oddc.device)
      } \
      --state-dir /var/lib/gjallar-usbtrust \
      --request-timeout 30s \
      ${lib.optionalString (cfg.tpmHandle != null) "--tpm-handle ${lib.escapeShellArg cfg.tpmHandle}"} \
      ${lib.optionalString cfg.enforce "--enforce"} \
      --usbguard-binary ${pkgs.usbguard}/bin/usbguard
  '';
in
{
  options.services.gjallar-usbtrust = {
    enable = lib.mkOption {
      type = lib.types.bool;
      default = settings.usbguardEnable or false;
      description = "Run the machine-local USB trust broker and observation backend.";
    };
    enforce = lib.mkOption {
      type = lib.types.bool;
      default = settings.usbTrustEnforce or false;
      description = "Apply derived USB decisions. Leave false until enrollment and hardware validation are complete.";
    };
    tpmHandle = lib.mkOption {
      type = lib.types.nullOr lib.types.str;
      default = if configuredTPMHandle == "" then null else configuredTPMHandle;
      example = "0x81000042";
      description = "Explicitly reserved TPM persistent P-256 signing handle; required for permanent trust.";
    };
  };

  config = lib.mkMerge [
    { services.udisks2.enable = true; }
    (lib.mkIf cfg.enable {
      assertions = [
        {
          assertion = config.oddc.device != null;
          message = "USB trust requires a selected ODDC model.";
        }
        {
          assertion = cfg.tpmHandle == null || builtins.match "^0x810[0-9a-fA-F]{5}$" cfg.tpmHandle != null;
          message = "USB trust TPM handle must be a reserved owner handle in 0x81000000..0x810fffff.";
        }
        {
          assertion = !cfg.enforce || cfg.tpmHandle != null;
          message = "USB trust enforcement requires a configured TPM signing handle; enroll and audit machine identities before enabling enforcement.";
        }
      ];
      users.groups.usb-trust-access = { };
      users.users.${settings.username}.extraGroups = [ "usb-trust-access" ];

      services.usbguard = {
        enable = true;
        dbus.enable = false;
        IPCAllowedUsers = lib.mkForce [ "root" ];
        IPCAllowedGroups = lib.mkForce [ ];
        rules = "";
        implicitPolicyTarget = if cfg.enforce then "block" else "allow";
        presentDevicePolicy = "apply-policy";
        insertedDevicePolicy = "apply-policy";
        presentControllerPolicy = "keep";
        restoreControllerDeviceState = true;
        deviceRulesWithPort = false;
      };

      systemd.services.usbguard.preStart = lib.mkBefore ''
        rm -f -- /var/lib/usbguard/IPCAccessControl.d/${lib.escapeShellArg settings.username}
      '';

      systemd.services.usbguard.serviceConfig.ReadWritePaths = lib.mkForce [
        "-/dev/shm"
        "-/tmp"
        "-/sys"
      ];

      systemd.services.usb-trust-broker = {
        description = "USB trust broker";
        wantedBy = [
          "multi-user.target"
          "usbguard.service"
        ];
        requires = [ "usbguard.service" ];
        bindsTo = [ "usbguard.service" ];
        partOf = [ "usbguard.service" ];
        after = [ "usbguard.service" ];
        path = [ pkgs.tpm2-tools ];
        serviceConfig = {
          Type = "simple";
          Environment = [ "TPM2TOOLS_TCTI=device:/dev/tpmrm0" ];
          User = "root";
          Group = "usb-trust-access";
          ExecStart = start;
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
      home-manager.users.${settings.username} = { pkgs, ... }: {
        systemd.user.services = lib.optionalAttrs (!(settings.endpointManagedDevice or false)) {
          usb-device-review = {
            Unit = {
              Description = "USB device review";
              After = [ "hyprland-session.target" ];
              PartOf = [ "hyprland-session.target" ];
            };
            Service = {
              ExecStart = "${pkgs.callPackage ../../pkgs/gjallarctl { }}/bin/gjallarctl usb review";
              Environment = [
                "PATH=${lib.makeBinPath [ pkgs.zenity pkgs.libnotify ]}"
                "SUDO_ASKPASS=${usbTrustAskpass}"
              ];
              Restart = "on-failure";
              RestartSec = 3;
            };
            Install.WantedBy = [ "hyprland-session.target" ];
          };
        };
      };
    })
  ];
}
