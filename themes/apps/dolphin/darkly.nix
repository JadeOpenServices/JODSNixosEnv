{ pkgs }:

{
  # Dolphin uses Darkly for widget rendering only. qtct/Noctalia remains the
  # palette authority, so live semantic colors are not duplicated here.
  styleName = "Darkly";
  package = pkgs.darkly;

  # Keep the plugin lookup version-aware through Nixpkgs' Qt metadata instead
  # of hardcoding lib/qt-6/plugins.
  qtPluginPath = "${pkgs.darkly}/${pkgs.qt6.qtbase.qtPluginPrefix}";
}
