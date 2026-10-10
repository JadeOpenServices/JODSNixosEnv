{
  config,
  osConfig ? { },
  pkgs,
  lib,
  ...
}:
let
  # Hyprland matches a client by the full path of its executable, so every
  # rule covers the whole store path of one package. Store paths are
  # read-only, so a user program cannot pose as one of these.
  inPackage = pkg: "${lib.escapeRegex (toString pkg)}/.*";
  portal = osConfig.programs.hyprland.portalPackage or pkgs.xdg-desktop-portal-hyprland;
  pluginPath = plugin: "${plugin}/lib/lib${plugin.pname}.so";

  rules = [
    "${inPackage portal}, screencopy, allow"
    "${inPackage portal}, cursorpos, allow"
    "${inPackage config.programs.hyprlock.package}, screencopy, allow"
  ]
  ++ lib.optionals config.programs.noctalia.enable [
    "${inPackage config.programs.noctalia.package}, screencopy, allow"
  ]
  ++ map (plugin: "${lib.escapeRegex (pluginPath plugin)}, plugin, allow") (
    builtins.filter lib.isDerivation config.wayland.windowManager.hyprland.plugins
  );

  # Hyprland keeps the rules it started with until the next login. It records
  # which ones at start, and activation compares that with the new set: a
  # Noctalia or hyprlock update moves their store paths, and until a new login
  # every screenshot of theirs would ask for permission.
  rulesId = builtins.hashString "sha256" (builtins.toJSON rules);
  startedRules = "gjallar/hyprland-permissions";
  recordRules = pkgs.writeShellScript "gjallar-record-permissions" ''
    set -eu
    mkdir -p "$XDG_RUNTIME_DIR/gjallar"
    echo ${rulesId} >"$XDG_RUNTIME_DIR/${startedRules}"
  '';
in
{
  # Without this any program in the session can read the screen through
  # wlr-screencopy without the user knowing. Clients without a rule get a
  # hyprland-dialog prompt; screen sharing for browsers and calls goes
  # through the portal and its own window picker.
  # https://wiki.hypr.land/Configuring/Advanced-and-Cool/Permissions/
  # Rules load only at Hyprland start, so a change needs a new login.
  wayland.windowManager.hyprland.settings = {
    ecosystem.enforce_permissions = true;
    permission = rules;
    exec-once = [ "${recordRules}" ];
  };

  home.activation.gjallarPermissionsNotice = lib.hm.dag.entryAfter [ "writeBoundary" ] ''
    runtime=/run/user/$(${pkgs.coreutils}/bin/id -u)
    if [ -r "$runtime/${startedRules}" ] &&
       [ "$(${pkgs.coreutils}/bin/cat "$runtime/${startedRules}")" != ${rulesId} ]; then
      run ${pkgs.coreutils}/bin/env DBUS_SESSION_BUS_ADDRESS=unix:path=$runtime/bus \
        ${pkgs.libnotify}/bin/notify-send \
        --app-name=GjallarOS --icon=system-log-out \
        "Log out to finish the update" \
        "Screen capture rules changed. Until you log in again, screenshots may ask for permission." ||
        true
    fi
  '';
}
