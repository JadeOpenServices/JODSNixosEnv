{
  config,
  pkgs,
  lib,
  hyprlandShellDetails,
  gjallarRun,
  settings,
  ...
}:
let
  shell = hyprlandShellDetails.binds;
  profile = builtins.fromJSON (builtins.readFile ./keybinds.json);

  commands = {
    volumeUp = shell.volumeUp;
    volumeDown = shell.volumeDown;

    brightnessUp = shell.brightnessUp;
    brightnessDown = shell.brightnessDown;

    lock = shell.lock;

    launcher = if shell.launcher == "fuzzel" then lib.getExe pkgs.fuzzel else shell.launcher;

    terminal = "${lib.getExe gjallarRun} ${lib.getExe pkgs.ghostty}";

    editor = "${lib.getExe gjallarRun} ${lib.getExe pkgs.${settings.preferredEditor}}";
    browser = "${lib.getExe gjallarRun} ${lib.getExe pkgs.${settings.preferredBrowser}}";

    fileManager = lib.getExe config._module.args.gjallarFileManager;

    gjallarAI = "gjallar-ai";

    libreoffice = "${lib.getExe gjallarRun} ${lib.getExe pkgs.libreoffice}";
    teams = "${lib.getExe gjallarRun} gjallar-teams";

    screenshot = shell.screenshot;

    micMute = shell.micMute;
    mediaNext = shell.mediaNext;
    mediaPrev = shell.mediaPrev;
    mediaToggle = shell.mediaToggle;
  };

  render =
    binding:
    let
      mods = binding.mods or "";
      isExec = binding.action == "exec";

      command =
        if isExec && builtins.hasAttr binding.command commands then
          commands.${binding.command}
        else
          binding.command or binding.action;

      args = binding.args or "";
    in
    if isExec then
      "${mods}, ${binding.key}, exec, ${command}${if args == "" then "" else " ${args}"}"
    else
      "${mods}, ${binding.key}, ${command}${if args == "" then "," else ", ${args}"}";

  # Commands that exist only when their app is selected.
  appOf = {
    fileManager = "dolphin";
    gjallarAI = "opencode";
    libreoffice = "libreoffice";
    teams = "teams";
  };

  available =
    binding:
    !(binding.action == "exec" && appOf ? ${binding.command or ""})
    || config.gjallar.apps.${appOf.${binding.command}}.enable;

  renderBindm = binding: "${binding.mods or ""}, ${binding.key}, ${binding.action}";

  # Hyprland matches keys without case and mods in any order, so
  # "$mod, L" and "$mod, l" are the same bind.
  bindId =
    binding:
    let
      mods = builtins.filter (m: m != "") (lib.splitString " " (lib.toUpper (binding.mods or "")));
    in
    "${lib.concatStringsSep " " (lib.sort lib.lessThan mods)}, ${lib.toUpper binding.key}";
  bindIds = map bindId (profile.bind ++ profile.binde);
  duplicateBinds = lib.unique (builtins.filter (id: lib.count (x: x == id) bindIds > 1) bindIds);
in
{
  assertions = [
    {
      assertion = duplicateBinds == [ ];
      message = "keybinds.json binds the same key twice: ${lib.concatStringsSep "; " duplicateBinds}";
    }
  ];

  wayland.windowManager.hyprland.settings = {
    "$mod" = "SUPER";

    bindm = map renderBindm profile.bindm;
    binde = map render (builtins.filter available profile.binde);
    bind = map render (builtins.filter available profile.bind);
  };
}
