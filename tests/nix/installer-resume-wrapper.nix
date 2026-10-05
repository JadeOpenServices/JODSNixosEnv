{ lib, options, ... }:
{
  config = lib.mkIf (!(options ? gjallar && options.gjallar ? installerResume)) {
    systemd.services.installer-resume = {
      description = "Resume the GjallarOS installer after a planned or interrupted reboot";
      wantedBy = [ "multi-user.target" ];
      wants = [ "network-online.target" ];
      after = [ "local-fs.target" "network-online.target" ];
      path = [ "/run/wrappers" "/run/current-system/sw" ];
      unitConfig = {
        ConditionPathExists = [ "|/var/lib/gjallarOS/installer-resume/pending.json" "|/var/lib/gjallarOS/installer-resume/active.json" ];
        ConditionFileIsExecutable = "/var/lib/gjallarOS/installer-resume/gjallar-installer";
      };
      serviceConfig = {
        Type = "oneshot";
        UMask = "0077";
        TimeoutStartSec = 0;
        ExecStart = "/var/lib/gjallarOS/installer-resume/gjallar-installer --resume-transaction /var/lib/gjallarOS/installer-resume";
      };
    };
  };
}
