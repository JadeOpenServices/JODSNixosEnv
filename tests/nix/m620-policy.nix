{
  nixpkgs,
  system,
  graphicsModule,
}:
let
  settings = {
    graphicsVendor = "nvidia";
    graphicsDeviceId = "13b4";
    graphicsDriverBranch = "legacy_580";
    graphicsType = "hybrid";
    graphicsCompute = true;
    graphicsBusId = "PCI:1:0:0";
    graphicsIntegratedBusId = "PCI:0:2:0";
  };

  evaluated = nixpkgs.lib.nixosSystem {
    inherit system;
    modules = [
      graphicsModule
      {
        system.stateVersion = "26.05";

        nixpkgs.config.allowUnfreePredicate =
          pkg:
          builtins.elem (nixpkgs.lib.getName pkg) [
            "nvidia-x11"
          ];
      }
    ];
    specialArgs = {
      inherit settings;
    };
  };

  config = evaluated.config;
in
assert
  config.hardware.nvidia.package.drvPath
  == config.boot.kernelPackages.nvidiaPackages.legacy_580.drvPath;
assert config.hardware.nvidia.open == false;
assert config.hardware.nvidia.prime.offload.enable;
assert config.hardware.nvidia.prime.intelBusId == "PCI:0:2:0";
assert config.hardware.nvidia.prime.nvidiaBusId == "PCI:1:0:0";
nixpkgs.legacyPackages.${system}.runCommand "gjallar-m620-policy-check" { } ''
  touch "$out"
''
