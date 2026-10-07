{
  config,
  lib,
  pkgs,
  settings,
  ...
}:

let
  gpuContract = lib.attrByPath [
    "policy"
    "virtualization"
    "gpu"
  ] { } config.oddc.resolved;

  passthrough = gpuContract.passthrough or { };
  passthroughMode = passthrough.mode or "disabled";
  passthroughValidated = passthrough.validated or false;

  passthroughAllowed =
    passthroughValidated
    && builtins.elem passthroughMode [
      "sriov"
      "mdev"
      "vfio-exclusive"
    ];
in
{
  imports = [
    ./module.nix
    ./spice.nix
  ];

  config = lib.mkIf config.gjallar.apps.nemu.enable {
    warnings = lib.optional (passthroughMode != "disabled" && !passthroughAllowed) ''
      ODDC GPU passthrough policy is not validated. Physical GPU passthrough
      is disabled and host GPU ownership is preserved.
    '';

    environment.etc."nemu/gpu-policy.json".text = builtins.toJSON {
      genericAcceleration = "auto";
      physicalPassthrough = if passthroughAllowed then passthroughMode else "disabled";
      source = if passthroughAllowed then "oddc-validated" else "safe-default";
    };

    environment.systemPackages = with pkgs; [
      virt-viewer
      tigervnc
      qemu
      samba
    ];

    programs.nemu = {
      package = pkgs._nemu;
      enable = true;
      vhostNetGroup = "nemu-vhost";
      macvtapGroup = "nemu-vhost";
      users = {
        ${settings.username} = {
          autoAddVeth = false;
          autoStartDaemon = false;
        };
      };
    };
  };
}
