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

    mkdir -p "$WINEPREFIX"

    # Fully grant write and execute permissions on prefix
    chmod -R 755 "$WINEPREFIX" 2>/dev/null || true
    chmod -R +w "$WINEPREFIX" 2>/dev/null || true

    # Kill lingering wineserver processes to release file locks
    ${winePackage}/bin/wineserver -k || true

    # Phase 1: Initialize prefix cleanly
    ${winePackage}/bin/wineboot -u
    ${winePackage}/bin/wineserver -w || true

    # Re-apply write permissions after store files are linked
    chmod -R +w "$WINEPREFIX"

    # Set driver registry keys & disable DirectComposition hardware acceleration
    ${winePackage}/bin/wine reg add "HKCU\\Software\\Wine\\X11 Driver" /v ClientSideWithRender /t REG_SZ /d "N" /f || true
    ${winePackage}/bin/wine reg add "HKCU\\Software\\Wine\\X11 Driver" /v DirectColor /t REG_SZ /d "Y" /f || true
    ${winePackage}/bin/wine reg add "HKCU\\Software\\Wine\\X11 Driver" /v Decorated /t REG_SZ /d "Y" /f || true
    ${winePackage}/bin/wine reg add "HKCU\\Software\\Wine\\X11 Driver" /v Managed /t REG_SZ /d "Y" /f || true
    ${winePackage}/bin/wine reg add "HKCU\\Software\\Wine\\Direct3D" /v csmt /t REG_DWORD /d 1 /f || true
    ${winePackage}/bin/wine reg add "HKCU\\Software\\Wine\\Direct3D" /v MaxVersionGL /t REG_DWORD /d 40005 /f || true

    # Disable DirectComposition / DComp acceleration to prevent stub infinite loops
    ${winePackage}/bin/wine reg add "HKCU\\Software\\Microsoft\\Avalon.Graphics" /v DisableHWAcceleration /t REG_DWORD /d 1 /f || true

    # Phase 2: System DLLs, Fonts, and VC++ Redistributable
    ${pkgs.winetricks}/bin/winetricks -q corefonts comctl32 gdiplus vcrun2022 || true
    ${winePackage}/bin/wineserver -w || true
    chmod -R +w "$WINEPREFIX"

    # Phase 3: DirectX components & DComp override support
    ${pkgs.winetricks}/bin/winetricks -q dxvk vkd3d d3dcompiler_43 d3dcompiler_47 d3dx9 dcomp || true
    ${winePackage}/bin/wineserver -w || true
    chmod -R +w "$WINEPREFIX"

    # Phase 4: .NET Framework & Modern .NET Runtimes
    ${pkgs.winetricks}/bin/winetricks -q dotnet48 dotnetdesktop6 dotnetdesktop8 || true
    ${winePackage}/bin/wineserver -w || true
  '';

  wineWin11 = pkgs.writeShellScriptBin "wine-win11" ''
    set -euo pipefail
    prefix="''${WINEPREFIX:-$HOME/.local/share/wine/win11}"
    export WINEPREFIX="$prefix"
    export WINEARCH=win64
    export WINEESYNC=1
    export WINEFSYNC=1
    export DXVK_LOG_LEVEL=none

    # Rendering stability & surface initialization overrides
    export DXVK_ASYNC=0
    export DXVK_ENABLE_NVAPI=0
    export DXVK_FRAME_RATE=0
    export WINE_FULLSCREEN_FSR=0

    # DXVK state caching & performance optimizations
    export DXVK_STATE_CACHE=1
    export DXVK_STATE_CACHE_PATH="''${XDG_CACHE_HOME:-$HOME/.cache}/dxvk"
    mkdir -p "$DXVK_STATE_CACHE_PATH"

    # Staging & Memory Optimizations
    export STAGING_SHARED_MEMORY=1
    export WINE_LARGE_ADDRESS_AWARE=1

    # VKD3D Feature Level (DirectX 12 feature support)
    export VKD3D_FEATURE_LEVEL=12_1

    # Audio latency stabilization (PipeWire / PulseAudio)
    export PULSE_LATENCY_MSEC=60

    # Disable WPF hardware acceleration to bypass DirectComposition hangs
    export WPF_DISABLE_HARDWARE_ACCELERATION=1

    # Mute harmless fixme logs and enforce native DXVK / D3D / DComp overrides
    export WINEDEBUG="-fixme"
    export WINEDLLOVERRIDES="winhttp=n,b;d3dcompiler_47=n,b;d3dcompiler_43=n,b;gdiplus=n,b;dxgi=n,b;d3d11=n,b;d3d10core=n,b;d3d9=n,b;dcomp=n,b"

    # Auto-initialize if prefix directory does NOT exist or lacks initialization marker
    if [[ ! -d "$prefix" ]] || [[ ! -f "$prefix/.gjallar-wine-initialized" ]]; then
      mkdir -p "$prefix"
      ${wineInit}/bin/wine-win11-init
      touch "$prefix/.gjallar-wine-initialized"
    fi

    # Execute wine binary if arguments are passed, otherwise stop after init check
    if [ "$#" -gt 0 ]; then
      exec ${winePackage}/bin/wine "$@"
    fi
  '';

  # Desktop file to register system-wide executable handlers
  wineWin11Desktop = pkgs.makeDesktopItem {
    name = "wine-win11";
    desktopName = "Wine Windows 11 Wrapper";
    comment = "Runs Windows executables through modern customized Wine-Staging prefix";
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
    if settings.system == "aarch64-linux" && builtins.hasAttr "fex" pkgs then
      [ pkgs.fex ]
    else if settings.system == "aarch64-linux" && builtins.hasAttr "fex-emu" pkgs then
      [ pkgs.fex-emu ]
    else
      [ ];
in
{
  imports = [ ./spice.nix ] ++ lib.optional settings.nemuEnable ./nemu;

  environment.systemPackages =
    with pkgs;
    [
      podman-compose
      distrobox
      libvirt
      qemu

      winePackage
      wineInit
      wineWin11
      wineWin11Desktop
      winetricks
      dxvk
      vkd3d
      cabextract
      p7zip
      ntfs3g

      # Host UI & launcher runtimes
      umu-launcher
      dotnet-runtime_8
      webkitgtk_4_1
    ]
    ++ fexPackages;

  # Associate .exe, .msi, and shortcut files system-wide
  xdg.mime.defaultApplications = {
    "application/x-ms-dos-executable" = "wine-win11.desktop";
    "application/x-msi" = "wine-win11.desktop";
    "application/x-ms-shortcut" = "wine-win11.desktop";
  };

  # Automatically run pre-initialization on login if prefix doesn't exist
  systemd.user.services.wine-win11-preinit = {
    description = "Pre-initialize Wine Win11 Prefix";
    wantedBy = [ "default.target" ];
    serviceConfig = {
      Type = "oneshot";
      ExecStart = "${wineWin11}/bin/wine-win11";
      RemainAfterExit = true;
    };
  };

  boot.kernelParams = lib.optionals settings.nemuGpuPassthrough (
    [
      (if settings.graphicsVendor == "intel" then "intel_iommu=on" else "amd_iommu=on")
    ]
    ++ lib.optional (settings.nemuGpuIds != [ ]) (
      "vfio-pci.ids=" + lib.concatStringsSep "," settings.nemuGpuIds
    )
  );

  boot.initrd.kernelModules = lib.optionals settings.nemuGpuPassthrough [
    "vfio"
    "vfio_pci"
    "vfio_iommu_type1"
  ];

  environment.etc."nemu/gpu-passthrough.conf" = lib.mkIf settings.nemuGpuPassthrough {
    text = lib.concatStringsSep "\n" settings.nemuGpuIds + "\n";
  };

  # Never expose a root Docker daemon. `dockerEnable` provides Docker-compatible
  # commands backed by the calling user's rootless Podman storage and namespace.
  virtualisation.docker.enable = false;

  virtualisation.podman = {
    enable = true;
    dockerCompat = settings.dockerEnable;
    defaultNetwork.settings.dns_enabled = true;
  };
}
