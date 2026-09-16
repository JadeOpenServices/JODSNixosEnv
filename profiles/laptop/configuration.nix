{
  lib,
  settings,
  ...
}:
{
  imports = [
    ../../system/base.nix
    ../../generated/hardware.nix

    ../../system/hardware/desktop/mouse.nix
    ../../system/security/laptop/firewall.nix
    ../../system/hardware/laptop/battery.nix
    ../../system/hardware/laptop/boot.nix

    ../../oddc/nixos/modules/default.nix
  ];

  oddc.device =
    let
      model = settings.oddcModel or "";
    in
    if model == "" then null else model;

  users.users.${settings.username}.extraGroups = [
    "networkmanager"
    "kvm"
    "vhost"
    "usb"
  ]
  ++ lib.optionals (!settings.endpointManagedDevice) [ "wheel" ];

  system.stateVersion = "23.11";
}
