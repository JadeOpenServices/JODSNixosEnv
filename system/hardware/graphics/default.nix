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

          substituteInPlace "$target" \
            --replace-fail \
              '    strncpy(buf, current->comm, len - 1);' \
              '    strscpy(buf, current->comm, len);'

          nvswitch=""
          for candidate in \
            nvidia/linux_nvswitch.c \
            kernel/nvidia/linux_nvswitch.c \
            kernel-open/nvidia/linux_nvswitch.c
          do
            if [ -f "$candidate" ]; then
              nvswitch="$candidate"
              break
            fi
          done

          if [ -z "$nvswitch" ]; then
            echo "ERROR: NVIDIA 580 linux_nvswitch.c not found" >&2
            false
          fi

          substituteInPlace "$nvswitch" \
            --replace-fail \
              '    strncpy(regkey_val, regkey_val_start, regkey_val_len);' \
              '    memcpy(regkey_val, regkey_val_start, regkey_val_len);' \
            --replace-fail \
              '    return strncpy(dest, src, length);' \
              '    NvLength copy_len = strnlen(src, length);
    memcpy(dest, src, copy_len);
    if (copy_len < length)
    {
        memset(dest + copy_len, 0, length - copy_len);
    }
    return dest;'


          uvm_pmm=""
          for candidate in \
            nvidia-uvm/uvm_pmm_gpu.c \
            kernel/nvidia-uvm/uvm_pmm_gpu.c \
            kernel-open/nvidia-uvm/uvm_pmm_gpu.c
          do
            if [ -f "$candidate" ]; then
              uvm_pmm="$candidate"
              break
            fi
          done

          if [ -z "$uvm_pmm" ]; then
            echo "ERROR: NVIDIA 580 uvm_pmm_gpu.c not found" >&2
            false
          fi

          substituteInPlace "$uvm_pmm" \
            --replace-fail \
              '                strncpy(chunk_split_cache[level].name, "uvm_gpu_chunk_t", sizeof(chunk_split_cache[level].name) - 1);' \
              '                strscpy(chunk_split_cache[level].name, "uvm_gpu_chunk_t", sizeof(chunk_split_cache[level].name));'


          modeset=""
          for candidate in \
            nvidia-modeset/nvidia-modeset-linux.c \
            kernel/nvidia-modeset/nvidia-modeset-linux.c \
            kernel-open/nvidia-modeset/nvidia-modeset-linux.c
          do
            if [ -f "$candidate" ]; then
              modeset="$candidate"
              break
            fi
          done

          if [ -z "$modeset" ]; then
            echo "ERROR: NVIDIA 580 nvidia-modeset-linux.c not found" >&2
            false
          fi

          substituteInPlace "$modeset" \
            --replace-fail \
              '    return strncpy(dest, src, n);' \
              '    size_t copy_len = strnlen(src, n);
    memcpy(dest, src, copy_len);
    if (copy_len < n)
    {
        memset(dest + copy_len, 0, n - copy_len);
    }
    return dest;'
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
