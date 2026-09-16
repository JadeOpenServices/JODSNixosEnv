{
  lib,
  settings,
  ...
}:
{
  imports = [
    ../../system/base.nix
    ../../generated/hardware.nix

    ../../system/hardware/thinkpad/boot.nix
    ../../system/security/laptop/firewall.nix
    ../../system/hardware/laptop/battery.nix
  ];

  users.users.${settings.username}.extraGroups = [
    "networkmanager"
  ]
  ++ lib.optionals (!settings.endpointManagedDevice) [ "wheel" ];

  system.stateVersion = "25.05";
}
