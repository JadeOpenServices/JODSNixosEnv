{ config, lib, ... }:
{
  services.fprintd.enable = true;
  services.gnome.gnome-keyring.enable = true;

  # A fingerprint cannot provide the login password needed to decrypt GNOME
  # Keyring. Require a password for new login sessions so Chromium/Edge/Teams
  # receive an unlocked Secret Service. Fingerprints remain available after
  # login for the lock screen and privilege elevation.
  security.pam.services = {
    greetd = {
      fprintAuth = false;
      enableGnomeKeyring = true;
    };
    hyprlock.fprintAuth = true;
    login = {
      fprintAuth = false;
      enableGnomeKeyring = true;
    };
    sudo = {
      fprintAuth = true;

      # After fingerprint failure/timeout, start a fresh password
      # conversation instead of reusing the fprint PAM token.
      rules.auth.unix.args = lib.mkForce [ "likeauth" ];
    };
    swaylock.fprintAuth = true;
  };

  security.polkit.extraConfig = lib.mkAfter ''
    polkit.addRule(function(action, subject) {
      if (action.id == "net.reactivated.fprint.device.enroll" &&
          subject.active && subject.local) {
        return polkit.Result.YES;
      }
    });
  '';

  environment.etc."systemd/system-sleep/restart-fprintd" = {
    mode = "0755";
    text = ''
      #!/bin/sh
      if [ "$1" = post ]; then
        ${lib.getExe' config.systemd.package "systemctl"} try-restart fprintd.service >/dev/null 2>&1 || true
      fi
    '';
  };
}
