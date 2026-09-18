{
  lib,
  pkgs,
  settings,
  ...
}:
let
  enable = settings.nextcloudEnable or false;
  host = settings.nextcloudHost or "";
  localRoot = settings.nextcloudLocalRoot or "";

  setup = pkgs.writeShellScriptBin "gjallar-nextcloud-setup" ''
    exec ${lib.getExe pkgs.nextcloud-client} \
      --overrideserverurl ${lib.escapeShellArg host} \
      --overridelocaldir ${lib.escapeShellArg localRoot}
  '';
in
{
  # Nextcloud owns remote/local synchronization and mutable account state.
  #
  # Credentials, app passwords and account tokens remain exclusively in the
  # client's credential store and must never enter Nix, generated state or Git.
  home.packages = lib.optionals enable [
    pkgs.nextcloud-client
    setup
  ];

  # Friendly first-run entry point using only non-secret installer intent.
  xdg.desktopEntries = lib.mkIf enable {
    nextcloud-setup = {
      name = "Nextcloud Setup";
      genericName = "Cloud synchronization setup";
      comment = "Connect this GjallarOS user to Nextcloud";
      exec = "${setup}/bin/gjallar-nextcloud-setup";
      icon = "Nextcloud";
      terminal = false;
      categories = [
        "Network"
        "FileTransfer"
      ];
    };
  };

  # Authentication remains interactive. Background synchronization begins
  # only after the Nextcloud client has created its account configuration.
  systemd.user.services.nextcloud-sync = lib.mkIf enable {
    Unit = {
      Description = "Nextcloud desktop synchronization";
      After = [ "graphical-session.target" ];
      ConditionPathExists = "%h/.config/Nextcloud/nextcloud.cfg";
    };

    Service = {
      Type = "simple";
      ExecStart = "${pkgs.nextcloud-client}/bin/nextcloud --background";

      Restart = "on-failure";
      RestartSec = "5s";

      NoNewPrivileges = true;
      PrivateTmp = true;
      RestrictSUIDSGID = true;
      LockPersonality = true;
    };

    Install.WantedBy = [
      "graphical-session.target"
    ];
  };
}
