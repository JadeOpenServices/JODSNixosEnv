{
  config,
  lib,
  pkgs,
  settings,
  ...
}:

let
  managed = settings.endpointManagedDevice or false;

  # This may only be enabled through signed JODS-managed configuration.
  # Unmanaged installer presets intentionally do not expose it.
  jodsEnrollmentAllowed =
    settings.jodsFingerprintEnrollmentAllowed or false;

  fingerprintPresent =
    builtins.hasAttr "primary" (
      lib.attrByPath [
        "oddc"
        "resolved"
        "hardware"
        "security"
        "fingerprint"
      ] { } config
    );

  fingerprintState = pkgs.writeShellScript "gjallar-fingerprint-state" ''
    set -eu

    user="''${1:-}"

    if [ -z "$user" ] ||
       ! ${pkgs.getent}/bin/getent passwd "$user" >/dev/null; then
      echo invalid
      exit 0
    fi

    # fprintd keeps per-user enrollment metadata below /var/lib/fprint.
    # Do not call back into fprintd from its own polkit authorization path.
    if [ -d "/var/lib/fprint/$user" ] &&
       ${pkgs.findutils}/bin/find \
         "/var/lib/fprint/$user" \
         -type f -print -quit 2>/dev/null |
       ${pkgs.gnugrep}/bin/grep -q .; then
      echo enrolled
    else
      echo initial
    fi
  '';
in
lib.mkMerge [
  {
    assertions = [
      {
        assertion = !jodsEnrollmentAllowed || managed;
        message =
          "jodsFingerprintEnrollmentAllowed is valid only on a JODS-managed device.";
      }
      {
        assertion = !jodsEnrollmentAllowed || fingerprintPresent;
        message =
          "jodsFingerprintEnrollmentAllowed requires fingerprint hardware in the resolved ODDC device.";
      }
    ];
  }

  (lib.mkIf fingerprintPresent {
    services.fprintd.enable = true;

    security.pam.services = {
      greetd.fprintAuth = false;
      hyprlock.fprintAuth = true;
      login.fprintAuth = false;

      sudo = {
        fprintAuth = true;

        # After fingerprint failure/timeout, start a fresh password
        # conversation instead of reusing the fprint PAM token.
        rules.auth.unix.args = lib.mkForce [ "likeauth" ];
      };

      # Polkit authorization must use a password. Otherwise an already enrolled
      # fingerprint could authorize adding another fingerprint.
      polkit-1 = {
        fprintAuth = false;
        rules.auth.unix.args = lib.mkForce [ "likeauth" ];
      };

      swaylock.fprintAuth = true;
    };

    security.polkit.extraConfig = lib.mkBefore ''
      polkit.addRule(function(action, subject) {
        if (action.id != "net.reactivated.fprint.device.enroll") {
          return polkit.Result.NOT_HANDLED;
        }

        if (!subject.active || !subject.local) {
          return polkit.Result.NO;
        }

        if (${if managed then "true" else "false"}) {
          // JODS owns enrollment policy. A local UI cannot grant itself access.
          if (!${if jodsEnrollmentAllowed then "true" else "false"}) {
            return polkit.Result.NO;
          }

          return polkit.Result.AUTH_ADMIN;
        }

        var state =
          polkit.spawn(["${fingerprintState}", subject.user]);

        if (state.trim() == "enrolled") {
          // Do not use *_KEEP: every later add/delete gets a fresh admin prompt.
          return polkit.Result.AUTH_ADMIN;
        }

        return polkit.Result.AUTH_SELF;
      });
    '';
  })
]
