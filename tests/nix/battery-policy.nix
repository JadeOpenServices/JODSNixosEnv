{ pkgs }:

let
  laptop = builtins.fromJSON (
    builtins.readFile ../../oddc/catalog/entities/classes/laptop.json
  );

  framework = builtins.fromJSON (
    builtins.readFile ../../oddc/catalog/entities/models/framework/laptop-13-amd-ryzen-7040.json
  );

  batteryAvailable =
    laptop.data.capabilities.battery or false;

  chargeThresholdsSupported =
    framework.data.capabilities.chargeThresholds or false;

  powerProfilesDaemonEnabled =
    laptop.data.policy.power.powerProfilesDaemon.enable or false;
in
assert batteryAvailable;
assert !chargeThresholdsSupported;
assert powerProfilesDaemonEnabled;
pkgs.runCommand "gjallar-framework-battery-policy-check" { } ''
  touch "$out"
''
