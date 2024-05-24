{ config, pkgs, ... }:
{
    networking.wg-quick.interfaces = {
        wg0 = {
            autostart = false;
            address = [ "198.51.100.3/32" ];
            privateKeyFile = "/etc/wireguard/private.key";
            dns = [ "198.51.100.1" ];
            mtu = 1450;
            peers = [{
                publicKey = "0catAJY1uisr7cL8qu9t3Nwo6nlf6Go7rByyi0iWCC0=";
                presharedKeyFile = "/etc/wireguard/preshared.key";
                allowedIPs = [ "0.0.0.0/0" ];
                endpoint = "203.0.113.6:51820";
                persistentKeepalive = 15;
            }];
        };
    };

    services.v2raya.enable = true;
}
