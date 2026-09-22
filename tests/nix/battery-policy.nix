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

  batteryProtection = laptop.data.policy.power.batteryProtection;
  chargeThresholdPolicy = laptop.data.policy.power.chargeThresholds;
in
assert batteryAvailable;
assert !chargeThresholdsSupported;
assert powerProfilesDaemonEnabled;
assert batteryProtection.lowWarningPercent == 10;
assert batteryProtection.criticalWarningPercent == 5;
assert batteryProtection.shutdownPercent == 2;
assert chargeThresholdPolicy.startPercent == 75;
assert chargeThresholdPolicy.endPercent == 95;
pkgs.runCommand "gjallar-framework-battery-policy-check" { } ''
  touch "$out"
''
