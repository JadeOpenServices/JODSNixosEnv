{
  config,
  pkgs,
  lib,
  hyprlandShellDetails,
  gjallarRun,
  gjallarApps,
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
    || builtins.elem appOf.${binding.command} gjallarApps;

  renderBindm = binding: "${binding.mods or ""}, ${binding.key}, ${binding.action}";
in
{

  wayland.windowManager.hyprland.settings = {
    "$mod" = "SUPER";

    bindm = map renderBindm profile.bindm;
    binde = map render (builtins.filter available profile.binde);
    bind = map render (builtins.filter available profile.bind);
  };
}
