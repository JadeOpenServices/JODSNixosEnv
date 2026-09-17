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

  # Generic virtualization never takes ownership of a host GPU.
  # Physical passthrough requires an explicit validated ODDC contract.
  warnings = lib.optional (
    passthroughMode != "disabled" && !passthroughAllowed
  ) ''
    ODDC GPU passthrough policy is not validated. Physical GPU passthrough
    is disabled and host GPU ownership is preserved.
  '';

  environment.etc."nemu/gpu-policy.json".text = builtins.toJSON {
    genericAcceleration = "auto";
    physicalPassthrough =
      if passthroughAllowed then passthroughMode else "disabled";
    source =
      if passthroughAllowed then "oddc-validated" else "safe-default";
  };

  environment.systemPackages = with pkgs; [
    virt-viewer
    tigervnc
    qemu
    samba
  ];

  imports = [
    ./module.nix
  ];

  programs.nemu = {
    package = pkgs._nemu;
    enable = true;
    vhostNetGroup = "vhost";
    macvtapGroup = "vhost";
    usbGroup = "usb";
    users = {
      ${settings.username} = {
        # Install Nemu without starting its daemon or creating veth
        # devices during boot. Users can start it on demand once a
        # VM definition exists; this keeps normal boots clean.
        autoAddVeth = false;
        autoStartDaemon = false;
        # autoStartVMs = [ "Win11" ];
      };
    };
  };
}
