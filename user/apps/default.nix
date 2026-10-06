{ lib, settings, ... }:

let
  # Optional apps by ID. `settings.apps` selects them; without it an
  # install keeps every app it had before the selection existed.
  catalog = {
    btop = ./btop;
    dolphin = ./dolphin.nix;
    drawio = ./drawio.nix;
    fastfetch = ./fastfetch;
    gimp = ./gimp.nix;
    git = ./git.nix;
    keepassxc = ./keepassxc.nix;
    libreoffice = ./libreoffice.nix;
    nextcloud = ./nextcloud.nix;
    opencode = ./opencode.nix;
    plane = ./plane.nix;
    qbittorrent = ./qbittorrent.nix;
    rust = ./rust.nix;
    teams = ./teams.nix;
    vlc = ./vlc.nix;
    wine = ./wine.nix;
  };

  selected = settings.apps or (builtins.attrNames catalog);

  unknown = builtins.filter (id: !(catalog ? ${id})) selected;
in
{
  imports =
    assert lib.assertMsg (unknown == [ ]) "unknown apps: ${toString unknown}";
    [
      # Base: the window managers launch ghostty, and fingerprint follows
      # the hardware ODDC resolved.
      ./ghostty.nix
      ./fingerprint.nix
    ]
    ++ map (id: catalog.${id}) selected;

  # The selected app IDs, for modules that wire up optional apps.
  _module.args.gjallarApps = selected;
}
