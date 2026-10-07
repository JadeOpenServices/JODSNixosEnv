# Legacy adapter: generated state still names system apps by their old
# settings; map them onto the catalogue's gjallar.apps.<id>.enable.
{ lib, settings, ... }:

{
  gjallar.apps = {
    ai.enable = lib.mkDefault (settings.aiEnable or false);
    containers.enable = lib.mkDefault (settings.containersEnable or false);
    nemu.enable = lib.mkDefault (settings.nemuEnable or false);
    # null: state from before the installer asked about Tailscale; it stays on.
    tailscale.enable = lib.mkDefault (
      (settings.tailscale or null) == null || (settings.tailscale.enable or true)
    );
  };
}
