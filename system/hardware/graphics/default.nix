{ config, lib, pkgs, settings, ... }:
let
    vendor = settings.graphicsVendor;
    hybrid = settings.graphicsType == "hybrid";
in {
    services.xserver.enable = true;
    hardware.graphics = {
        enable = true;
        enable32Bit = true;
    };
    hardware.enableRedistributableFirmware = true;
    hardware.wirelessRegulatoryDatabase.enable = true;

    boot.initrd.kernelModules = lib.concatLists [
        (lib.optional (vendor == "amd") "amdgpu")
        (lib.optional (vendor == "intel") "i915")
    ];
    services.xserver.videoDrivers =
        if vendor == "nvidia" then [ "nvidia" ] else [ "modesetting" ];

    environment.systemPackages = with pkgs; lib.optionals (vendor == "intel") [
        intel-media-driver
        intel-vaapi-driver
    ];

    hardware.nvidia = lib.mkIf (vendor == "nvidia") {
        open = true;
        modesetting.enable = true;
        powerManagement.enable = true;
        prime = lib.mkIf hybrid {
            offload.enable = true;
            intelBusId = settings.graphicsIntegratedBusId;
            nvidiaBusId = settings.graphicsBusId;
        };
    };

    hardware.amdgpu.opencl.enable = lib.mkIf (vendor == "amd" && settings.graphicsCompute) true;
    hardware.graphics.extraPackages = lib.mkIf (vendor == "amd" && settings.graphicsCompute) (with pkgs; [
        rocmPackages.clr
        rocmPackages.clr.icd
    ]);
}
