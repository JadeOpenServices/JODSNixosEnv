{
  config,
  lib,
  pkgs,
  settings,
  ...
}:
let
  vendor = settings.graphicsVendor;
  driverBranch = settings.graphicsDriverBranch or "stable";
  legacy580 = vendor == "nvidia" && driverBranch == "legacy_580";
  hybrid = settings.graphicsType == "hybrid";

  selectedNvidiaPackage =
    if legacy580 then
      config.boot.kernelPackages.nvidiaPackages.legacy_580
    else
      config.boot.kernelPackages.nvidiaPackages.stable;

  nvidiaPackage =
    if vendor == "nvidia" && lib.hasPrefix "580." selectedNvidiaPackage.version then
      selectedNvidiaPackage.overrideAttrs (old: {
        postPatch = (if old.postPatch == null then "" else old.postPatch) + ''
          target=""

          for candidate in \
            nvidia/os-interface.c \
            kernel/nvidia/os-interface.c \
            kernel-open/nvidia/os-interface.c
          do
            if [ -f "$candidate" ]; then
              target="$candidate"
              break
            fi
          done

          if [ -z "$target" ]; then
            echo "ERROR: NVIDIA 580 os-interface.c not found" >&2
            false
          fi

          if ! grep -q '<linux/string.h>' "$target"; then
            sed -i '/#include <linux\/sys_soc.h>/a #include <linux/string.h>' "$target"
          fi
        '';
      })
    else
      selectedNvidiaPackage;
in
{
  services.xserver.enable = true;
  hardware.graphics = {
    enable = true;
    enable32Bit = true;
  };
  hardware.enableRedistributableFirmware = true;
  hardware.wirelessRegulatoryDatabase = true;

  boot.initrd.kernelModules = lib.concatLists [
    (lib.optional (vendor == "amd") "amdgpu")
    (lib.optional (vendor == "intel") "i915")
  ];
  services.xserver.videoDrivers = if vendor == "nvidia" then [ "nvidia" ] else [ "modesetting" ];

  environment.systemPackages =
    with pkgs;
    lib.optionals (vendor == "intel") [
      intel-media-driver
      intel-vaapi-driver
    ];

  hardware.nvidia = lib.mkIf (vendor == "nvidia") {
    package = nvidiaPackage;
    open = !legacy580;
    modesetting.enable = true;
    powerManagement.enable = true;
    prime = lib.mkIf hybrid {
      offload.enable = true;
      intelBusId = settings.graphicsIntegratedBusId;
      nvidiaBusId = settings.graphicsBusId;
    };
  };

  hardware.amdgpu.opencl.enable = lib.mkIf (vendor == "amd" && settings.graphicsCompute) true;
  hardware.graphics.extraPackages = lib.mkIf (vendor == "amd" && settings.graphicsCompute) (
    with pkgs;
    [
      rocmPackages.clr
      rocmPackages.clr.icd
    ]
  );
}
