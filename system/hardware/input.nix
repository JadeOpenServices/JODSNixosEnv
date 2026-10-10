{ lib, options, settings, ... }:
let
  # ODDC turns on iio-sensor-proxy itself for a model with a proven
  # orientation sensor (oddc.sensors.orientation). Older ODDC pins have no
  # such option; there the installer's flag still decides.
  oddcOrientation = lib.hasAttrByPath [ "oddc" "sensors" "orientation" "present" ] options;
in
{
  services.libinput.enable = true;

  hardware.sensor.iio.enable = lib.mkIf (!oddcOrientation) (
    settings.orientationSensorEnable or false
  );
}
