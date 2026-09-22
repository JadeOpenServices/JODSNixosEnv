{
  nixpkgs,
  system,
  graphicsModule,
}:
let
  settings = {
    graphicsBusId = "PCI:1:0:0";
    graphicsIntegratedBusId = "PCI:0:2:0";
  };

  oddcResolved = (import ../../oddc/lib/registry.nix { lib = nixpkgs.lib; }).resolveEntity "model/hp/zbook-x2-g4";

  evaluated = nixpkgs.lib.nixosSystem {
    inherit system;
    modules = [
      graphicsModule
      (
        { lib, ... }:
        {
          options.oddc.resolved = lib.mkOption {
            type = lib.types.attrs;
            default = { };
          };
          config.oddc.resolved = oddcResolved.resolved;
          config.system.stateVersion = "26.05";

          config.nixpkgs.config.allowUnfreePredicate =
            pkg:
            builtins.elem (nixpkgs.lib.getName pkg) [
              "nvidia-x11"
            ];
        }
      )
    ];
    specialArgs = {
      inherit settings;
    };
  };

  config = evaluated.config;
in
assert
  config.hardware.nvidia.package.version
  == config.boot.kernelPackages.nvidiaPackages.legacy_580.version;
assert config.hardware.nvidia.open == false;
assert config.hardware.nvidia.prime.offload.enable;
assert config.hardware.nvidia.prime.intelBusId == "PCI:0:2:0";
assert config.hardware.nvidia.prime.nvidiaBusId == "PCI:1:0:0";
nixpkgs.legacyPackages.${system}.runCommand "gjallar-m620-policy-check" { } ''
  touch "$out"
''
