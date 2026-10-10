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
    permission = [
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
  };
}
