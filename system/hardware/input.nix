{ settings, ... }:
{
  services.libinput.enable = true;

  hardware.sensor.iio.enable = settings.orientationSensorEnable or false;
}
