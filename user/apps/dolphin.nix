{
  config,
  lib,
  pkgs,
  settings,
  gjallarRun,
  ...
}:
let
  dolphinTheme = import ../../themes/apps/dolphin/darkly.nix { inherit pkgs; };
  dolphinNextcloudIcons = import ../../themes/apps/dolphin/nextcloud-icons.nix;

  fileManager = pkgs.writeShellApplication {
    name = "file-manager";
    text = ''
      if [ "$#" -eq 0 ]; then
        set -- "$HOME"
      fi

      # Darkly is Dolphin-local. qtct remains active and continues supplying
      # the live Noctalia/KColorScheme palette.
      export QT_STYLE_OVERRIDE=${lib.escapeShellArg dolphinTheme.styleName}
      export QT_PLUGIN_PATH=${lib.escapeShellArg dolphinTheme.qtPluginPath}''${QT_PLUGIN_PATH:+:$QT_PLUGIN_PATH}

      exec ${lib.getExe gjallarRun} ${lib.getExe pkgs.kdePackages.dolphin} "$@"
    '';
  };
  ini = pkgs.formats.ini { };
  defaults = ini.generate "dolphin-defaults" {
    General = {
      AlwaysShowTabBar = true;
      BrowseThroughArchives = true;
      FilterBar = true;
      GlobalViewProps = true;
      OpenExternallyCalledFolderInNewTab = false;
      RememberOpenedTabs = false;
      ShowFullPath = true;
      ShowToolTips = true;
      ShowZoomSlider = true;
      SortingChoice = 0;
    };
    ContextMenu.ShowCopyMoveMenu = true;
    InformationPanel = {
      previewsShown = true;
      previewsAutoPlay = false;
    };
    # Keep remote browsing from fetching file content merely for thumbnails.
    PreviewSettings = {
      EnableRemoteFolderThumbnail = false;
      MaximumRemoteSize = 0;
    };
  };
  editorDesktop =
    {
      vscodium = "codium.desktop";
      vscode = "code.desktop";
    }
    .${settings.preferredEditor} or null;
in
{
  _module.args.gjallarFileManager = fileManager;

  home.packages = with pkgs.kdePackages; [
    dolphinTheme.package
    dolphin
    dolphin-plugins
    ark
    kio-extras
    kio-fuse
    kdegraphics-thumbnailers
    ffmpegthumbs
    konsole
    kfind
    filelight
    fileManager
  ];

  # Stylix supplies the desktop's dark Kvantum palette and fonts.
  qt.qt6ctSettings.Appearance.icon_theme = lib.mkDefault settings.themeDetails.icons;

  # Seed once: Dolphin owns its live settings so UI changes survive rebuilds.
  home.activation.dolphinDefaults = lib.hm.dag.entryAfter [ "writeBoundary" ] ''
    dolphin_config=${lib.escapeShellArg "${config.xdg.configHome}/dolphinrc"}
    if [ ! -e "$dolphin_config" ] && [ ! -L "$dolphin_config" ]; then
      run ${pkgs.coreutils}/bin/install -D -m 0600 ${defaults} "$dolphin_config"
    fi
  '';

  # Seed the managed Nextcloud root into Dolphin Places without
  # taking ownership of the user's live Places configuration.
  # Do not restore an ever-growing collection of old tabs, and do
  # not funnel externally opened folders into the current Dolphin window.
  home.activation.dolphinSessionPolicy = lib.hm.dag.entryAfter [ "dolphinDefaults" ] ''
    dolphin_config=${lib.escapeShellArg "${config.xdg.configHome}/dolphinrc"}
    write_config=${lib.getExe' pkgs.kdePackages.kconfig "kwriteconfig6"}

    remember_opened_tabs=
    externally_called_folder=
    in_general=false

    if [ -r "$dolphin_config" ]; then
      while IFS= read -r line || [ -n "$line" ]; do
        case "$line" in
          '[General]')
            in_general=true
            continue
            ;;
          '['*']')
            if [ "$in_general" = true ]; then
              break
            fi
            continue
            ;;
        esac

        if [ "$in_general" = true ]; then
          case "$line" in
            RememberOpenedTabs=*)
              remember_opened_tabs="''${line#RememberOpenedTabs=}"
              ;;
            OpenExternallyCalledFolderInNewTab=*)
              externally_called_folder="''${line#OpenExternallyCalledFolderInNewTab=}"
              ;;
          esac
        fi
      done < "$dolphin_config"
    fi

    if [ "$remember_opened_tabs" != "false" ]; then
      run "$write_config" --file dolphinrc --group General --key RememberOpenedTabs false
    fi

    if [ "$externally_called_folder" != "false" ]; then
      run "$write_config" --file dolphinrc --group General --key OpenExternallyCalledFolderInNewTab false
    fi
  '';

  home.activation.dolphinNextcloudPlace = lib.mkIf (settings.nextcloudEnable or false) (
    lib.hm.dag.entryAfter [ "dolphinDefaults" "dolphinSessionPolicy" ] ''
      places=${lib.escapeShellArg "${config.xdg.dataHome}/user-places.xbel"}
    nextcloud_uri="file://$HOME/Nextcloud"

    if [ -f "$places" ] &&
       ! ${pkgs.gnugrep}/bin/grep -Fq "href=\"$nextcloud_uri\"" "$places"
    then
      tmp="$places.gjallar-nextcloud.$$"
      base="$(${pkgs.coreutils}/bin/date +%s)"
      n=0

      while ${pkgs.gnugrep}/bin/grep -Fq "<ID>$base/$n</ID>" "$places"
      do
        n=$((n + 1))
      done

      nextcloud_id="$base/$n"

      ${pkgs.gnused}/bin/sed \
        "/<bookmark href=\"remote:\\/\">/i\\
 <bookmark href=\"$nextcloud_uri\">\\
  <title>Nextcloud</title>\\
  <info>\\
   <metadata owner=\"http://freedesktop.org\">\\
    <bookmark:icon name=\"${dolphinNextcloudIcons.places}\"/>\\
   </metadata>\\
   <metadata owner=\"http://www.kde.org\">\\
    <ID>$nextcloud_id</ID>\\
   </metadata>\\
  </info>\\
 </bookmark>" \
        "$places" > "$tmp"

      ${pkgs.coreutils}/bin/chmod --reference="$places" "$tmp"
        ${pkgs.coreutils}/bin/mv "$tmp" "$places"
      fi
    ''
  );

  xdg.desktopEntries."org.kde.dolphin" = {
    name = "Dolphin";
    genericName = "File Manager";
    comment = "Browse files, devices, archives and network locations";
    exec = "${lib.getExe fileManager} %U";
    icon = "system-file-manager";
    terminal = false;
    categories = [
      "Qt"
      "KDE"
      "System"
      "FileManager"
    ];
    mimeType = [ "inode/directory" ];
  };

  xdg.mimeApps = {
    enable = true;
    associations.added."inode/directory" = [ "org.kde.dolphin.desktop" ];
    defaultApplications = {
      "inode/directory" = [ "org.kde.dolphin.desktop" ];
    }
    // lib.optionalAttrs (editorDesktop != null) (
      lib.genAttrs [
        "text/plain"
        "text/markdown"
        "application/json"
        "application/javascript"
        "application/xml"
        "application/yaml"
        "text/yaml"
      ] (_: [ editorDesktop ])
    );
  };

  programs.zsh.shellAliases.fm = "${lib.getExe fileManager} .";
}
