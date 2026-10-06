{
  config,
  lib,
  pkgs,
  settings,
  ...
}:

let
  winePackage = pkgs.wineWow64Packages.staging;

  wineInit = pkgs.writeShellScriptBin "wine-win11-init" ''
    set -euo pipefail

    export WINEPREFIX="''${WINEPREFIX:-$HOME/.local/share/wine/win11}"
    export WINEARCH=win64
    export WINEDEBUG="-all"

    unset WINEDLLOVERRIDES

    install -d -m 0700 "$WINEPREFIX"

    ${winePackage}/bin/wineserver -k || true

    ${winePackage}/bin/wineboot -u
    ${winePackage}/bin/wineserver -w || true

    ${winePackage}/bin/wine reg add \
      "HKCU\\Software\\Wine\\X11 Driver" \
      /v ClientSideWithRender /t REG_SZ /d "N" /f || true

    ${winePackage}/bin/wine reg add \
      "HKCU\\Software\\Wine\\X11 Driver" \
      /v DirectColor /t REG_SZ /d "Y" /f || true

    ${winePackage}/bin/wine reg add \
      "HKCU\\Software\\Wine\\X11 Driver" \
      /v Decorated /t REG_SZ /d "Y" /f || true

    ${winePackage}/bin/wine reg add \
      "HKCU\\Software\\Wine\\X11 Driver" \
      /v Managed /t REG_SZ /d "Y" /f || true

    ${winePackage}/bin/wine reg add \
      "HKCU\\Software\\Wine\\Direct3D" \
      /v csmt /t REG_DWORD /d 1 /f || true

    ${winePackage}/bin/wine reg add \
      "HKCU\\Software\\Wine\\Direct3D" \
      /v MaxVersionGL /t REG_DWORD /d 40005 /f || true

    ${winePackage}/bin/wine reg add \
      "HKCU\\Software\\Microsoft\\Avalon.Graphics" \
      /v DisableHWAcceleration /t REG_DWORD /d 1 /f || true

    ${pkgs.winetricks}/bin/winetricks \
      -q corefonts comctl32 gdiplus vcrun2022 || true
    ${winePackage}/bin/wineserver -w || true

    ${pkgs.winetricks}/bin/winetricks \
      -q dxvk vkd3d d3dcompiler_43 d3dcompiler_47 d3dx9 dcomp || true
    ${winePackage}/bin/wineserver -w || true

    ${pkgs.winetricks}/bin/winetricks \
      -q dotnet48 dotnetdesktop6 dotnetdesktop8 || true
    ${winePackage}/bin/wineserver -w || true

    touch "$WINEPREFIX/.gjallar-wine-initialized"
  '';

  wineWin11 = pkgs.writeShellScriptBin "wine-win11" ''
    set -euo pipefail

    prefix="''${WINEPREFIX:-$HOME/.local/share/wine/win11}"

    export WINEPREFIX="$prefix"
    export WINEARCH=win64
    export WINEESYNC=1
    export WINEFSYNC=1
    export DXVK_LOG_LEVEL=none
    export DXVK_ASYNC=0
    export DXVK_ENABLE_NVAPI=0
    export DXVK_FRAME_RATE=0
    export WINE_FULLSCREEN_FSR=0
    export DXVK_STATE_CACHE=1
    export DXVK_STATE_CACHE_PATH="''${XDG_CACHE_HOME:-$HOME/.cache}/dxvk"
    export STAGING_SHARED_MEMORY=1
    export WINE_LARGE_ADDRESS_AWARE=1
    export VKD3D_FEATURE_LEVEL=12_1
    export PULSE_LATENCY_MSEC=60
    export WPF_DISABLE_HARDWARE_ACCELERATION=1
    export WINEDEBUG="-fixme"
    export WINEDLLOVERRIDES="winhttp=n,b;d3dcompiler_47=n,b;d3dcompiler_43=n,b;gdiplus=n,b;dxgi=n,b;d3d11=n,b;d3d10core=n,b;d3d9=n,b;dcomp=n,b"

    mkdir -p "$DXVK_STATE_CACHE_PATH"

    if [[ ! -d "$prefix" ]] ||
       [[ ! -f "$prefix/.gjallar-wine-initialized" ]]; then
      mkdir -p "$(dirname "$prefix")"

      exec 9>"$prefix.gjallar-init.lock"
      ${pkgs.util-linux}/bin/flock 9

      if [[ ! -f "$prefix/.gjallar-wine-initialized" ]]; then
        ${wineInit}/bin/wine-win11-init
      fi
    fi

    if [[ "$#" -gt 0 ]]; then
      exec ${winePackage}/bin/wine "$@"
    fi
  '';

  wineDesktop = pkgs.makeDesktopItem {
    name = "wine-win11";
    desktopName = "Wine Windows 11 Wrapper";
    comment = "Run Windows executables through GjallarOS Wine";
    exec = "${wineWin11}/bin/wine-win11 %f";
    terminal = false;

    mimeTypes = [
      "application/x-ms-dos-executable"
      "application/x-msi"
      "application/x-ms-shortcut"
      "application/x-wine-extension-msp"
    ];

    categories = [ "Utility" ];
  };

  fexPackages =
    if settings.system == "aarch64-linux" &&
       builtins.hasAttr "fex" pkgs then
      [ pkgs.fex ]
    else if settings.system == "aarch64-linux" &&
            builtins.hasAttr "fex-emu" pkgs then
      [ pkgs.fex-emu ]
    else
      [ ];
in
lib.mkIf config.gjallar.apps.wine.enable {
  home.packages =
    [
      winePackage
      wineInit
      wineWin11
      wineDesktop

      pkgs.winetricks
      pkgs.dxvk
      pkgs.vkd3d
      pkgs.cabextract
      pkgs.p7zip
    ]
    ++ fexPackages;

  xdg.mimeApps = {
    enable = true;

    defaultApplications = {
      "application/x-ms-dos-executable" = [ "wine-win11.desktop" ];
      "application/x-msi" = [ "wine-win11.desktop" ];
      "application/x-ms-shortcut" = [ "wine-win11.desktop" ];
      "application/x-wine-extension-msp" = [ "wine-win11.desktop" ];
    };
  };
}
