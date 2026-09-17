{ inputs, config, pkgs, lib, settings, gjallarRun, ... }:
let
  editor = lib.getExe pkgs.${settings.preferredEditor};
  colors = config.lib.stylix.colors;
  roles = {
    primary = "#${colors.base0D}";
    on_primary = "#${colors.base00}";
    secondary = "#${colors.base0B}";
    tertiary = "#${colors.base0E}";
    error = "#${colors.base08}";
    surface = "#${colors.base00}";
    on_surface = "#${colors.base05}";
    outline = "#${colors.base03}";
    surface_container_lowest = "#${colors.base01}";
  };
  fallback = text: builtins.replaceStrings
    (map (name: "{{colors.${name}.default.hex}}") (builtins.attrNames roles))
    (builtins.attrValues roles)
    text;
  theme = ''
    [app]
    overall = { bg = "{{colors.surface.default.hex}}" }

    [mgr]
    cwd = { fg = "{{colors.primary.default.hex}}" }
    hovered = { fg = "{{colors.on_primary.default.hex}}", bg = "{{colors.primary.default.hex}}", bold = true }
    preview_hovered = { fg = "{{colors.on_primary.default.hex}}", bg = "{{colors.primary.default.hex}}" }
    find_keyword = { fg = "{{colors.tertiary.default.hex}}", bold = true }
    find_position = { fg = "{{colors.secondary.default.hex}}" }
    marker_copied = { fg = "{{colors.secondary.default.hex}}", bg = "{{colors.secondary.default.hex}}" }
    marker_cut = { fg = "{{colors.error.default.hex}}", bg = "{{colors.error.default.hex}}" }
    marker_marked = { fg = "{{colors.tertiary.default.hex}}", bg = "{{colors.tertiary.default.hex}}" }
    marker_selected = { fg = "{{colors.primary.default.hex}}", bg = "{{colors.primary.default.hex}}" }
    border_style = { fg = "{{colors.outline.default.hex}}" }

    [indicator]
    parent = { fg = "{{colors.on_primary.default.hex}}", bg = "{{colors.primary.default.hex}}" }
    current = { fg = "{{colors.on_primary.default.hex}}", bg = "{{colors.primary.default.hex}}", bold = true }
    preview = { fg = "{{colors.on_primary.default.hex}}", bg = "{{colors.primary.default.hex}}" }

    [tabs]
    active = { fg = "{{colors.on_primary.default.hex}}", bg = "{{colors.primary.default.hex}}", bold = true }
    inactive = { fg = "{{colors.on_surface.default.hex}}", bg = "{{colors.surface_container_lowest.default.hex}}" }

    [mode]
    normal_main = { fg = "{{colors.on_primary.default.hex}}", bg = "{{colors.primary.default.hex}}", bold = true }
    normal_alt = { fg = "{{colors.on_surface.default.hex}}", bg = "{{colors.surface_container_lowest.default.hex}}" }
    select_main = { fg = "{{colors.surface.default.hex}}", bg = "{{colors.secondary.default.hex}}", bold = true }
    select_alt = { fg = "{{colors.on_surface.default.hex}}", bg = "{{colors.surface_container_lowest.default.hex}}" }
    unset_main = { fg = "{{colors.surface.default.hex}}", bg = "{{colors.error.default.hex}}", bold = true }
    unset_alt = { fg = "{{colors.on_surface.default.hex}}", bg = "{{colors.surface_container_lowest.default.hex}}" }

    [status]
    overall = { fg = "{{colors.on_surface.default.hex}}", bg = "{{colors.surface.default.hex}}" }
    perm_type = { fg = "{{colors.primary.default.hex}}" }
    perm_read = { fg = "{{colors.secondary.default.hex}}" }
    perm_write = { fg = "{{colors.tertiary.default.hex}}" }
    perm_exec = { fg = "{{colors.error.default.hex}}" }
    perm_sep = { fg = "{{colors.outline.default.hex}}" }
    progress_label = { fg = "{{colors.on_surface.default.hex}}", bold = true }
    progress_normal = { fg = "{{colors.primary.default.hex}}", bg = "{{colors.surface_container_lowest.default.hex}}" }
    progress_error = { fg = "{{colors.error.default.hex}}", bg = "{{colors.surface_container_lowest.default.hex}}" }

    [which]
    mask = { bg = "{{colors.surface.default.hex}}" }
    cand = { fg = "{{colors.primary.default.hex}}" }
    rest = { fg = "{{colors.on_surface.default.hex}}" }
    desc = { fg = "{{colors.secondary.default.hex}}" }
    separator_style = { fg = "{{colors.outline.default.hex}}" }

    [input]
    border = { fg = "{{colors.primary.default.hex}}" }
    title = { fg = "{{colors.primary.default.hex}}" }
    value = { fg = "{{colors.on_surface.default.hex}}" }
    selected = { fg = "{{colors.on_primary.default.hex}}", bg = "{{colors.primary.default.hex}}" }

    [pick]
    border = { fg = "{{colors.primary.default.hex}}" }
    active = { fg = "{{colors.on_primary.default.hex}}", bg = "{{colors.primary.default.hex}}" }
    inactive = { fg = "{{colors.on_surface.default.hex}}" }

    [confirm]
    border = { fg = "{{colors.primary.default.hex}}" }
    title = { fg = "{{colors.primary.default.hex}}" }
    btn_yes = { fg = "{{colors.on_primary.default.hex}}", bg = "{{colors.primary.default.hex}}" }
    btn_no = { fg = "{{colors.on_surface.default.hex}}", bg = "{{colors.surface_container_lowest.default.hex}}" }

    [help]
    on = { fg = "{{colors.primary.default.hex}}" }
    run = { fg = "{{colors.secondary.default.hex}}" }
    desc = { fg = "{{colors.on_surface.default.hex}}" }
    hovered = { fg = "{{colors.on_primary.default.hex}}", bg = "{{colors.primary.default.hex}}", bold = true }
    footer = { fg = "{{colors.on_surface.default.hex}}", bg = "{{colors.surface_container_lowest.default.hex}}" }
  '';
  init = ''
    require("sduf"):setup({
      filled_bg = "{{colors.primary.default.hex}}",
      filled_bg_warn = "{{colors.tertiary.default.hex}}",
      filled_bg_danger = "{{colors.error.default.hex}}",
      unfilled_bg = "{{colors.surface_container_lowest.default.hex}}",
      text_fg = "{{colors.on_surface.default.hex}}",
      error_fg = "{{colors.error.default.hex}}",
    })
  '';
  gjallarFileManager = pkgs.writeShellApplication {
    name = "gjallar-file-manager";

    text = ''
      target="''${1:-$HOME}"

      exec ${gjallarRun}/bin/gjallar-run \
        ${lib.getExe pkgs.ghostty} \
        -e ${lib.getExe config.programs.yazi.package} \
        "$target"
    '';
  };

in
{
  programs.yazi = {
    enable = true;
    enableZshIntegration = true;
    shellWrapperName = "y";
    package = pkgs.symlinkJoin {
      name = "yazi-gjallaros-${pkgs.yazi.version}";
      paths = [ pkgs.yazi ];
      nativeBuildInputs = [ pkgs.makeWrapper ];
      postBuild = ''
        wrapProgram "$out/bin/yazi" \
          --set EDITOR ${lib.escapeShellArg editor} \
          --set VISUAL ${lib.escapeShellArg editor} \
          --prefix PATH : ${lib.makeBinPath [ pkgs.coreutils ]}
      '';
      meta.mainProgram = "yazi";
    };
    settings = {
      mgr = {
        sort_by = "natural";
        sort_sensitive = false;
        sort_dir_first = true;
        linemode = "size";
        ratio = [ 1 4 3 ];
      };
      open.prepend_rules = [
        { mime = "text/*"; use = "edit"; }
        { mime = "application/json"; use = "edit"; }
        { mime = "application/javascript"; use = "edit"; }
        { mime = "application/xml"; use = "edit"; }
        { mime = "application/yaml"; use = "edit"; }
      ];
    };
    plugins = {
      sduf = builtins.toPath inputs.yazi-disk-space.outPath;
      mount = pkgs.yaziPlugins.mount;
    };

    keymap.mgr.prepend_keymap = [
      {
        on = [ "M" ];
        run = "plugin mount";
        desc = "Mount, unmount or eject removable media";
      }
    ];
  };

  _module.args.gjallarFileManager = gjallarFileManager;

  stylix.targets.yazi.enable = false;

  home.packages = with pkgs; [
    glib ffmpeg poppler-utils exiftool zoxide
    file fd ripgrep fzf chafa wl-clipboard gjallarFileManager
  ];

  # Session-wide file-manager route. GUI file chooser dialogs remain
  # portal-owned; opening a directory uses this GjallarOS application route.
  xdg.desktopEntries.gjallar-yazi = {
    name = "Yazi File Manager";
    genericName = "File Manager";
    comment = "Browse files with Yazi in Ghostty";
    exec = "${lib.getExe gjallarFileManager} %f";
    terminal = false;
    mimeType = [ "inode/directory" ];
  };

  xdg.mimeApps = {
    enable = true;
    associations.added."inode/directory" = [ "gjallar-yazi.desktop" ];
    defaultApplications."inode/directory" = [ "gjallar-yazi.desktop" ];
  };

  xdg.configFile."noctalia/templates/yazi.toml".text = theme;
  xdg.configFile."noctalia/templates/yazi-init.lua".text = init;

  home.activation.gjallarYaziThemeFallback = lib.hm.dag.entryAfter [ "writeBoundary" ] ''
    mkdir -p ${lib.escapeShellArg "${config.xdg.configHome}/yazi"}
    if [ ! -e ${lib.escapeShellArg "${config.xdg.configHome}/yazi/theme.toml"} ]; then
      cp ${pkgs.writeText "yazi-fallback-theme.toml" (fallback theme)} \
        ${lib.escapeShellArg "${config.xdg.configHome}/yazi/theme.toml"}
      chmod u+w ${lib.escapeShellArg "${config.xdg.configHome}/yazi/theme.toml"}
    fi
    if [ ! -e ${lib.escapeShellArg "${config.xdg.configHome}/yazi/init.lua"} ]; then
      cp ${pkgs.writeText "yazi-fallback-init.lua" (fallback init)} \
        ${lib.escapeShellArg "${config.xdg.configHome}/yazi/init.lua"}
      chmod u+w ${lib.escapeShellArg "${config.xdg.configHome}/yazi/init.lua"}
    fi
  '';
}
