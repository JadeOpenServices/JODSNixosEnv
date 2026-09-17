{ ... }:
{
  networking.networkmanager = {
    enable = true;

    # Avoid severe throughput/latency regressions seen with Wi-Fi power
    # saving while leaving normal suspend behavior intact.
    wifi.powersave = false;

    # Portable systems may have unused Ethernet adapters without carrier.
    # Keep their carrier wait short without changing Wi-Fi DHCP behavior.
    settings."device-gjallar-ethernet-fast-start" = {
      match-device = "type:ethernet";
      carrier-wait-timeout = 1000;
    };
  };

  # Keep network-online useful without allowing an unused interface to
  # stall normal boot for a full minute.
  systemd.services.NetworkManager-wait-online.environment.NM_ONLINE_TIMEOUT = "12";
}
