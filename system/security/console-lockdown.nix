{ lib, settings, ... }:
lib.mkIf (!(settings.consoleLoginEnable or false)) {
  # A text login on Ctrl+Alt+F2 sits next to the greeter and skips
  # everything the graphical session enforces. ChromeOS opens VT2 only in
  # developer mode
  # (https://www.chromium.org/chromium-os/chromiumos-design-docs/developer-shell-access/),
  # kiosk setups turn off autovt the same way
  # (https://manpages.debian.org/testing/systemd/logind.conf.5.en.html).
  # Deeper access goes through the maintenance boot entry, which keeps its
  # tty1 login. The installer resume unit writes to tty1 itself.
  services.logind.settings.Login = {
    NAutoVTs = 0;
    ReserveVT = 0;
  };
  systemd.services =
    lib.genAttrs
      [
        "getty@"
        "autovt@"
        "serial-getty@"
        "console-getty"
        "container-getty@"
      ]
      (_: {
        enable = false;
      });
}
