{ config, lib, pkgs, settings, ... }:

let
  winePackage = pkgs.wineWow64Packages.staging;

  wineWin11 = pkgs.writeShellScriptBin "wine-win11" ''
    set -euo pipefail
    prefix="''${WINEPREFIX:-$HOME/.local/share/wine/win11}"
    export WINEPREFIX="$prefix"
    export WINEARCH=win64
    export WINEESYNC=1
    export WINEFSYNC=1
    export DXVK_LOG_LEVEL=none

    mkdir -p "$prefix"

    if [[ ! -f "$prefix/.gjallar-wine-initialized" ]]; then
      ${winePackage}/bin/wineboot -u
      touch "$prefix/.gjallar-wine-initialized"
    fi

    exec ${winePackage}/bin/wine "$@"
  '';

  wineInit = pkgs.writeShellScriptBin "wine-win11-init" ''
    set -euo pipefail
    export WINEPREFIX="''${WINEPREFIX:-$HOME/.local/share/wine/win11}"
    export WINEARCH=win64

    ${winePackage}/bin/wineboot -u
    ${pkgs.winetricks}/bin/winetricks -q win11 dxvk vkd3d || true
  '';

  fexPackages =
    if settings.system == "aarch64-linux" && builtins.hasAttr "fex" pkgs
    then [ pkgs.fex ]
    else if settings.system == "aarch64-linux" && builtins.hasAttr "fex-emu" pkgs
    then [ pkgs.fex-emu ]
    else [];
in
{
  imports = [ ./spice.nix ] ++ lib.optional settings.nemuEnable ./nemu;

  environment.systemPackages = with pkgs; [
    docker-compose
    distrobox
    libvirt
    qemu

    winePackage
    wineInit
    wineWin11
    winetricks
    dxvk
    vkd3d
    cabextract
    p7zip
    ntfs3g
  ] ++ fexPackages;

  boot.kernelParams =
    lib.optionals settings.nemuGpuPassthrough (
      [
        (if settings.graphicsVendor == "intel"
         then "intel_iommu=on"
         else "amd_iommu=on")
      ]
      ++ lib.optional
        (settings.nemuGpuIds != [])
        ("vfio-pci.ids=" + lib.concatStringsSep "," settings.nemuGpuIds)
    );

  boot.initrd.kernelModules =
    lib.optionals settings.nemuGpuPassthrough [
      "vfio"
      "vfio_pci"
      "vfio_iommu_type1"
    ];

  environment.etc."nemu/gpu-passthrough.conf" =
    lib.mkIf settings.nemuGpuPassthrough {
      text = lib.concatStringsSep "\n" settings.nemuGpuIds + "\n";
    };

  virtualisation.docker.enable = settings.dockerEnable;

  virtualisation.podman.enable = true;

  virtualisation.podman.defaultNetwork.settings.dns_enabled = true;
}
