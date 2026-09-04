{ pkgs, ... }:
let
  enrollment = pkgs.writeShellScript "gjallar-fingerprint-enroll" ''
    set -eu
    marker="$HOME/.local/state/gjallar/fingerprint-enrollment-seen"
    mkdir -p "$(dirname "$marker")"
    [ ! -e "$marker" ] || exit 0

    if ! ${pkgs.fprintd}/bin/fprintd-list "$USER" >/dev/null 2>&1; then
      printf 'No supported fingerprint reader was found.\n'
      touch "$marker"
      printf 'Press Enter to close.\n'
      read -r _
      exit 0
    fi

    printf 'Set up a fingerprint for login and sudo? [Y/n] '
    read -r answer
    case "$answer" in
      n|N|no|NO|No) touch "$marker"; exit 0 ;;
    esac

    if ${pkgs.fprintd}/bin/fprintd-enroll "$USER"; then
      touch "$marker"
      printf '\nFingerprint enrolled. Your password remains available.\n'
    else
      printf '\nEnrollment failed. This prompt will return next login.\n' >&2
    fi
    printf 'Press Enter to close.\n'
    read -r _
  '';
  launcher = pkgs.writeShellScript "gjallar-fingerprint-enroll-launcher" ''
    marker="$HOME/.local/state/gjallar/fingerprint-enrollment-seen"
    [ ! -e "$marker" ] || exit 0
    exec ${pkgs.kitty}/bin/kitty --title 'GjallarOS fingerprint setup' -e ${enrollment}
  '';
in
{
  home.packages = [
    pkgs.fprintd
    pkgs.yad
  ];
  systemd.user.services.gjallar-fingerprint-enroll = {
    Unit = {
      Description = "Offer first-login fingerprint enrollment";
      After = [ "graphical-session.target" ];
      Wants = [ "graphical-session.target" ];
    };
    Service = {
      Type = "oneshot";
      ExecStart = launcher;
    };
    Install = {
      WantedBy = [ "default.target" ];
    };
  };
}
