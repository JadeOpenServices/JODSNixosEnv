{ config, pkgs, ... }:
{
  home.packages = [ pkgs.fprintd pkgs.yad ];
  systemd.user.services.alfheim-fingerprint-enroll = {
    Unit = {
      Description = "Offer first-login fingerprint enrollment";
      After = [ "graphical-session.target" ];
      PartOf = [ "graphical-session.target" ];
    };
    Service = {
      Type = "oneshot";
      ExecStart = pkgs.writeShellScript "alfheim-fingerprint-enroll" ''
        set -eu
        marker="$HOME/.config/alfheim/fingerprint-enrollment-seen"
        [ -e "$marker" ] && exit 0
        mkdir -p "$(dirname "$marker")"
        if ! ${pkgs.fprintd}/bin/fprintd-list "$USER" >/dev/null 2>&1; then
          touch "$marker"
          exit 0
        fi
        if ${pkgs.yad}/bin/yad --question --title='Fingerprint setup' --text='Set up a fingerprint for faster login?'; then
          ${pkgs.fprintd}/bin/fprintd-enroll || true
        fi
        touch "$marker"
      '';
    };
    Install = { WantedBy = [ "graphical-session.target" ]; };
  };
}
