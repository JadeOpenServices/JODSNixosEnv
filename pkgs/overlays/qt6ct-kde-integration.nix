final: prev:

let
  lib = prev.lib;

  qt6ctKdePatch = prev.fetchurl {
    url = "https://github.com/user-attachments/files/25205239/qt6ct-0.11.patch";
    hash = "sha256-2+HPQ7P1Yd8WFzGZP/u1vwxzVLUCEO5Lu0xcceFIOP8=";
  };

  patchedQt6Packages = prev.qt6Packages.overrideScope (
    scopeFinal: scopePrev: {
      qt6ct =
        assert lib.assertMsg (scopePrev.qt6ct.version == "0.11")
          "GjallarOS qt6ct KDE integration patch targets qt6ct 0.11; review or remove pkgs/overlays/qt6ct-kde-integration.nix before updating qt6ct.";
        scopePrev.qt6ct.overrideAttrs (old: {
          patches = (old.patches or [ ]) ++ [ qt6ctKdePatch ];

          buildInputs = (old.buildInputs or [ ]) ++ [
            scopeFinal.qtdeclarative
            prev.kdePackages.kconfig
            prev.kdePackages.kcolorscheme
            prev.kdePackages.kiconthemes
          ];
        });
    }
  );
in
{
  qt6Packages = patchedQt6Packages;

  # Keep both public package namespaces on the exact same patched derivation.
  kdePackages = prev.kdePackages.overrideScope (
    _scopeFinal: _scopePrev: {
      qt6ct = final.qt6Packages.qt6ct;
    }
  );
}
