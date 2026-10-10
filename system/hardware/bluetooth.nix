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

    # Both are BlueZ's defaults today; pinned so no later default or
    # override relaxes them. HID input only from bonded devices
    # (CVE-2023-45866), and no peer replaces a bond by Just Works re-pairing.
    settings.General = {
      ControllerMode = "dual";
      JustWorksRepairing = "never";
    };
    input.General.ClassicBondedOnly = true;
  };
}
