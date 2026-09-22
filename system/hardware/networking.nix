{ ... }:
{
  networking.networkmanager = {
    enable = true;

    wifi.powersave = false;

    settings."device-gjallar-ethernet-fast-start" = {
      match-device = "type:ethernet";
      carrier-wait-timeout = 1000;
    };
  };

  systemd.services.NetworkManager-wait-online.environment.NM_ONLINE_TIMEOUT = "12";
}
