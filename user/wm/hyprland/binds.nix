{ config, pkgs, lib, hyprlandShellDetails, ... }:
let
  shell = hyprlandShellDetails.binds;
  profile = builtins.fromJSON (builtins.readFile ./keybinds.json);
  commands = {
    volumeUp = shell.volumeUp; volumeDown = shell.volumeDown;
    brightnessUp = shell.brightnessUp; brightnessDown = shell.brightnessDown;
    lock = shell.lock;
    launcher = if shell.launcher == "fuzzel" then lib.getExe pkgs.fuzzel else shell.launcher;
    terminal = lib.getExe pkgs.kitty;
    screenshot = shell.screenshot;
    micMute = shell.micMute; mediaNext = shell.mediaNext; mediaPrev = shell.mediaPrev;
    mediaToggle = shell.mediaToggle;
  };
  render = binding:
    let
      mods = binding.mods or "";
      command = if binding.action == "exec"
        then if builtins.hasAttr binding.command commands
          then commands.${binding.command}
          else binding.command
        else binding.action;
      args = binding.args or "";
    in "${mods}, ${binding.key}, ${command}${if args == "" then "," else ", ${args}"}";
in {
  wayland.windowManager.hyprland.settings = {
    "$mod" = "SUPER";
    bindm = map render profile.bindm;
    binde = map render profile.binde;
    bind = map render profile.bind;
  };
}
