{
  lib,
  pkgs,
  settings,
  ...
}:
let
  usbguard = settings.usbguardEnable or false;
  diskReview = pkgs.writeShellApplication {
    name = "gjallar-disk-review";
    runtimeInputs = [
      pkgs.python3
      pkgs.util-linux
      pkgs.systemd
      pkgs.zenity
    ];
    text = ''
      exec python3 ${./removable-media-review.py}
    '';
  };
in
lib.mkIf usbguard {
  home.packages = [
    pkgs.zenity
  ];

  services.udiskie = {
    enable = true;
    automount = true;
    notify = true;
    tray = "auto";
  };

  systemd.user.services.removable-media-review =
    lib.mkIf (!(settings.endpointManagedDevice or false))
      {
        Unit = {
          Description = "Review unformatted USB disks";
          After = [ "graphical-session.target" ];
          PartOf = [ "graphical-session.target" ];
        };
        Service = {
          ExecStart = "${diskReview}/bin/gjallar-disk-review";
          Restart = "on-failure";
          RestartSec = 5;
        };
        Install.WantedBy = [ "graphical-session.target" ];
      };
}
