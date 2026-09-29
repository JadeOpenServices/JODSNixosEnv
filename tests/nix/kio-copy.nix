{ pkgs }:
pkgs.stdenv.mkDerivation {
  name = "gjallar-kio-copy-test";
  dontUnpack = true;
  nativeBuildInputs = [
    pkgs.pkg-config
    pkgs.kdePackages.wrapQtAppsHook
  ];
  buildInputs = [
    pkgs.kdePackages.qtbase
    pkgs.kdePackages.kio
  ];
  buildPhase = ''
    c++ ${../kio-copy.cpp} -o dolphin \
      $(pkg-config --cflags --libs Qt6Core) \
      -I${pkgs.kdePackages.kio.dev}/include/KF6/KIOCore \
      -I${pkgs.kdePackages.kio.dev}/include/KF6/KIO \
      -I${pkgs.kdePackages.kcoreaddons.dev}/include/KF6/KCoreAddons \
      -L${pkgs.kdePackages.kio}/lib -lKF6KIOCore \
      -L${pkgs.kdePackages.kcoreaddons}/lib -lKF6CoreAddons
  '';
  installPhase = "mkdir -p $out/bin; cp dolphin $out/bin/dolphin";
}
