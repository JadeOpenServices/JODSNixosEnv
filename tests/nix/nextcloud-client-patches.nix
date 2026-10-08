{ pkgs }:
let
  client = import ../../apps/nextcloud/client-source.nix { inherit (pkgs) fetchFromGitHub; };
in
pkgs.stdenvNoCC.mkDerivation {
  name = "gjallar-nextcloud-client-patches-check";
  inherit (client) src patches;
  dontConfigure = true;
  dontBuild = true;
  # OpenVFS files carry an "Inherited" pin. Checking only a file's own pin
  # kept every song copied out of an OnlineOnly folder downloaded for good
  # (2026-09-30); discovery must use the effective pin like xattr/suffix VFS.
  installPhase = ''
    f=src/libsync/vfs/openvfs/vfs_openvfs.cpp
    grep -Fq 'if (pin && *pin == PinState::OnlineOnly) {' "$f"
    grep -Fq 'if (pin && *pin == PinState::AlwaysLocal) {' "$f"
    # New local files have no placeholder attributes (default "Hydrated");
    # they must never be classified for dehydration.
    grep -Fq '} else if (attribs && attribs.state == ::OpenVFS::Constants::States::Hydrated) {' "$f"
    ! grep -Fq 'attribs.pinState == convertPinState(PinState::OnlineOnly)' "$f"
    # Reporting placeholders out of sync re-ran a no-op job on ~18k unchanged
    # files every sync run, a full core for hours (2026-10-05).
    grep -A6 'bool OpenVFS::isPlaceHolderInSync' "$f" | grep -Fq 'return true;'
    # qt6ct's KDE patch swaps Fusion for the missing org.kde.desktop style
    # during app construction, so "Add account" did nothing (2026-10-08).
    grep -A6 'OCC::Application app(argc, argv);' src/gui/main.cpp \
      | grep -Fq 'QQuickStyle::setStyle(qmlStyle);'
    touch $out
  '';
}
