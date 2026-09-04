{ config, lib, ... }:
{
  services.fprintd.enable = true;

  # Fingerprints supplement passwords; the normal PAM password rules remain.
  security.pam.services = {
    greetd.fprintAuth = true;
    hyprlock.fprintAuth = true;
    login.fprintAuth = true;
    sudo.fprintAuth = true;
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
