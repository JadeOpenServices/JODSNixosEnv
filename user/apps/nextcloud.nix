{
  lib,
  pkgs,
  settings,
  ...
}:
let
  enable = settings.nextcloudEnable or false;
  host = settings.nextcloudHost or "";

  client = pkgs.nextcloud-client;

  managedClient = pkgs.symlinkJoin {
    name = "gjallar-nextcloud-client";
    paths = [ client ];

    postBuild = ''
      rm -f "$out/bin/nextcloud"

      cat > "$out/bin/nextcloud" <<'SCRIPT'
#!${pkgs.runtimeShell}

config="''${XDG_CONFIG_HOME:-$HOME/.config}/Nextcloud/nextcloud.cfg"
local_root="$HOME/Nextcloud"

# Only a normal GUI launch may seed first-run wizard defaults.
# CLI operations such as --version must remain side-effect free.
if [ "$#" -eq 0 ]; then
  mkdir -p "$local_root"

  if [ ! -f "$config" ]; then
    # Only preselect the generic local cloud root.
    #
    # Do not force overrideServerUrl here. On Linux, the forced-server wizard
    # path derives VFS availability from isVfsEnabled in nextcloud.cfg instead
    # of directly using the runtime VFS backend. The normal wizard can detect
    # the installed suffix VFS plugin itself.
    ${lib.getExe client} \
      --overridelocaldir "$local_root"
  fi
fi

exec ${lib.getExe client} "$@"
SCRIPT

      chmod +x "$out/bin/nextcloud"

      # GjallarOS systemd owns background startup.
      rm -f "$out"/etc/xdg/autostart/*nextcloud*.desktop

      desktop="$out/share/applications/com.nextcloud.desktopclient.nextcloud.desktop"

      if [ -L "$desktop" ]; then
        source="$(readlink -f "$desktop")"
        rm "$desktop"
        cp "$source" "$desktop"
        chmod u+w "$desktop"
      fi

      if [ -f "$desktop" ]; then
        sed -i \
          "0,/^Exec=/{s|^Exec=.*|Exec=$out/bin/nextcloud|}" \
          "$desktop"
      fi
    '';
  };

  syncRunner = pkgs.writeShellScript "gjallar-nextcloud-sync" ''
    config="''${XDG_CONFIG_HOME:-$HOME/.config}/Nextcloud/nextcloud.cfg"

    # No account yet. Normal GUI startup owns enrollment.
    if [ ! -f "$config" ] ||
       ! grep -Eq '^[0-9]+\\url=' "$config"
    then
      echo "Nextcloud account setup is incomplete; background sync not started."
      exit 0
    fi

    # Never permit GjallarOS background startup to run an eager/full mirror.
    if grep -Eq \
      '^[0-9]+\\Folders(WithPlaceholders)?\\[0-9]+\\virtualFilesMode=off$' \
      "$config"
    then
      echo "Nextcloud background sync refused: virtual files are disabled." >&2
      exit 0
    fi

    if ! grep -Eq \
      '^[0-9]+\\Folders(WithPlaceholders)?\\[0-9]+\\virtualFilesMode=.+$' \
      "$config"
    then
      echo "Nextcloud background sync refused: no virtual-files folder is configured." >&2
      exit 0
    fi

    exec ${lib.getExe client} --background
  '';
in
{
  home.packages = lib.optionals enable [
    managedClient
  ];

  # Keep the cloud location visible even before account enrollment.
  home.activation.gjallarNextcloudRoot = lib.mkIf enable (
    lib.hm.dag.entryAfter [ "writeBoundary" ] ''
      ${pkgs.coreutils}/bin/mkdir -p "$HOME/Nextcloud"
    ''
  );

  # Nextcloud owns credentials, accounts and mutable sync state.
  # GjallarOS owns the generic local location and requires virtual files for
  # unattended/background synchronization.
  systemd.user.services.nextcloud-sync = lib.mkIf enable {
    Unit = {
      Description = "Nextcloud virtual-file synchronization";
      After = [ "graphical-session.target" ];
    };

    Service = {
      Type = "simple";
      ExecStart = syncRunner;

      # Nextcloud is a stateful, single-instance GUI client.
      # Do not immediately respawn it after an abnormal exit: that can race
      # Qt shared-memory teardown and turn one failure into a restart loop.
      Restart = "no";

      NoNewPrivileges = true;
      PrivateTmp = true;
      RestrictSUIDSGID = true;
      LockPersonality = true;
    };

    Install.WantedBy = [
      "graphical-session.target"
    ];
  };

  # Yazi integration: Nextcloud becomes a first-class location.
  programs.yazi.keymap.mgr.prepend_keymap = lib.mkAfter [
    {
      on = [
        "g"
        "n"
      ];
      run = "cd ~/Nextcloud";
      desc = "Go to Nextcloud";
    }
  ];

  programs.yazi.theme.icon.prepend_dirs = lib.mkBefore [
    {
      name = "Nextcloud";
      text = "";
    }
  ];

  programs.yazi.theme.icon.prepend_exts = lib.mkBefore [
    {
      name = "nextcloud";
      text = "󰇚";
    }
  ];

  # Linux suffix-VFS placeholders are explicitly opened through XDG.
  # The Nextcloud MIME handler performs hydration, then opens the real file.
  programs.yazi.settings.opener."nextcloud-hydrate" = [
    {
      run = "${pkgs.xdg-utils}/bin/xdg-open %s1";
      orphan = true;
      desc = "Download and open from Nextcloud";
    }
  ];

  programs.yazi.settings.open.prepend_rules = lib.mkBefore [
    {
      url = "*.nextcloud";
      use = "nextcloud-hydrate";
    }
  ];

  # Do not waste preview/preload workers on online-only placeholders.
  programs.yazi.settings.plugin.prepend_previewers = lib.mkBefore [
    {
      url = "*.nextcloud";
      run = "noop";
    }
  ];

  programs.yazi.settings.plugin.prepend_preloaders = lib.mkBefore [
    {
      url = "*.nextcloud";
      run = "noop";
    }
  ];
}
