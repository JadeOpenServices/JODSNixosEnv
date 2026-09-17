{
  lib,
  osConfig,
  pkgs,
  settings,
  ...
}:

let
  fingerprintPresent = builtins.hasAttr "primary" (
    lib.attrByPath [
      "oddc"
      "resolved"
      "hardware"
      "security"
      "fingerprint"
    ] { } osConfig
  );

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
      n|N|no|NO|No)
        touch "$marker"
        exit 0
        ;;
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

    exec ${lib.getExe pkgs.ghostty} \
      --title 'GjallarOS fingerprint setup' \
      -e ${enrollment}
  '';
in
lib.mkIf (fingerprintPresent && !(settings.endpointManagedDevice or false)) {
  home.packages = [
    pkgs.fprintd
    pkgs.yad
  ];

  systemd.user.services.gjallar-fingerprint-enroll = {
    Unit = {
      Description = "Offer first-login fingerprint enrollment";
      After = [ "graphical-session.target" ];
    };

    Service = {
      # Kitty stays open for user input. Type=oneshot makes Home Manager wait
      # for that window and eventually time out during a rebuild.
      Type = "exec";
      ExecStart = launcher;
    };

    Install.WantedBy = [ "graphical-session.target" ];
  };
}
