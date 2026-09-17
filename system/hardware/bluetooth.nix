{
  config,
  lib,
  pkgs,
  ...
}:

let
  bluetoothPresent =
    builtins.hasAttr "primary" (
      lib.attrByPath [
        "oddc"
        "resolved"
        "hardware"
        "network"
        "bluetooth"
      ] { } config
    );
in
lib.mkIf bluetoothPresent {
  environment.systemPackages = with pkgs; [
    bluez
    bluez-tools
  ];

  hardware.bluetooth = {
    enable = true;
    package = pkgs.bluez;

    settings.General.ControllerMode = "dual";
  };
}
