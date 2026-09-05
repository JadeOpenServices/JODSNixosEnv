{ config, pkgs,
  inputs,
  lib,
  ...
}:

let
  noctalia = lib.getExe config.programs.noctalia.package;
  # Central safe IPC boundary. Every shell action using `${ipc}` becomes
  # a silent no-op while Noctalia is disabled, starting, stopped or unavailable.
  ipc = "${pkgs.writeShellScript "gjallar-noctalia-ipc" ''
    if ${noctalia} msg status >/dev/null 2>&1; then
      ${noctalia} msg "$@" >/dev/null 2>&1 || true
    fi
    exit 0
  ''}";
in
{
  launchCommands = [
    "${noctalia}"
  ];

  binds = {
    launcher = "${ipc} panel-toggle launcher";
    controlCenter = "${ipc} panel-toggle control-center";

    volumeUp = "${ipc} volume-up 2";
    volumeDown = "${ipc} volume-down 2";
    brightnessUp = "${ipc} brightness-up current 5";
    brightnessDown = "${ipc} brightness-down current 5";
    lock = "${ipc} session lock";
    screenshot = "${ipc} screenshot-region";
    micMute = "${ipc} mic-mute";
    mediaNext = "${ipc} media next";
    mediaPrev = "${ipc} media previous";
    mediaToggle = "${ipc} media toggle";
  };
}
