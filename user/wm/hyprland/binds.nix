{ config, pkgs, lib, hyprlandShellDetails, settings, ... }:
let
  shell = hyprlandShellDetails.binds;
  profile = builtins.fromJSON (builtins.readFile ./keybinds.json);

  commands = {
    volumeUp = shell.volumeUp;
    volumeDown = shell.volumeDown;

    brightnessUp = shell.brightnessUp;
    brightnessDown = shell.brightnessDown;

    lock = shell.lock;

    launcher =
      if shell.launcher == "fuzzel"
      then lib.getExe pkgs.fuzzel
      else shell.launcher;

    terminal = lib.getExe pkgs.kitty;

    # Resolve selected applications from nixpkgs
    editor = lib.getExe pkgs.${settings.preferredEditor};
    browser = lib.getExe pkgs.${settings.preferredBrowser};

    fileManager = "${lib.getExe pkgs.kitty} -e ${lib.getExe pkgs.superfile}";

    # GjallarOS-AI
    gjallarAI = "opencode-local";

    # Applications
    libreoffice = lib.getExe pkgs.libreoffice;
    teams = "gjallar-teams";

    screenshot = shell.screenshot;

    micMute = shell.micMute;
    mediaNext = shell.mediaNext;
    mediaPrev = shell.mediaPrev;
    mediaToggle = shell.mediaToggle;
  };

  render = binding:
    let
      mods = binding.mods or "";
      isExec = binding.action == "exec";

      command =
        if isExec && builtins.hasAttr binding.command commands
        then commands.${binding.command}
        else binding.command or binding.action;

      args = binding.args or "";
    in
      if isExec
      then
        "${mods}, ${binding.key}, exec, ${command}${if args == "" then "" else " ${args}"}"
      else
        "${mods}, ${binding.key}, ${command}${if args == "" then "," else ", ${args}"}";

  renderBindm = binding:
    "${binding.mods or ""}, ${binding.key}, ${binding.action}";
in {
  wayland.windowManager.hyprland.settings = {
    "$mod" = "SUPER";

    bindm = map renderBindm profile.bindm;
    binde = map render profile.binde;
    bind = map render profile.bind;
  };
}