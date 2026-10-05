{ pkgs, ... }:

let
  applyLocalLanBypass = pkgs.writeShellScript "tailscale-local-lan-bypass" ''
    HOME_SUBNET="192.168.8.0/24"
    RULE_PRIORITY="2500"

    # Keep the local-LAN preference rule permanently present.
    #
    # suppress_prefixlength 0 is the fail-safe:
    #
    # - At home, main contains the real 192.168.8.0/24 connected route,
    #   so traffic stays directly on the LAN.
    #
    # - Away from home, main would otherwise return its default route.
    #   Prefix length 0 is suppressed, causing policy lookup to continue
    #   to Tailscale table 52 instead.
    #
    # Therefore a roaming Wi-Fi default gateway can never steal the
    # OpenJade home subnet.

    while ${pkgs.iproute2}/bin/ip -4 rule del \
      priority "$RULE_PRIORITY" \
      2>/dev/null
    do
      :
    done

    ${pkgs.iproute2}/bin/ip -4 rule add \
      priority "$RULE_PRIORITY" \
      to "$HOME_SUBNET" \
      lookup main \
      suppress_prefixlength 0

    ${pkgs.iproute2}/bin/ip -4 route flush cache

    echo "tailscale-local-lan-bypass: fail-safe home LAN preference active"
  '';
in
{
  services.tailscale.enable = true;

  # Re-evaluate the local-LAN preference whenever NetworkManager changes
  # connectivity. This keeps the 192.168.8.0/24 bypass active at home and
  # automatically yields to Tailscale's table 52 while roaming.
  networking.networkmanager.dispatcherScripts = [
    {
      source = pkgs.writeShellScript "tailscale-local-lan-bypass-dispatcher" ''
        case "$2" in
          up|down|dhcp4-change|dhcp6-change|connectivity-change)
            ${applyLocalLanBypass}
            ;;
        esac
      '';
      type = "basic";
    }
  ];

  systemd.services.tailscaled.postStart = ''
    ${applyLocalLanBypass}
  '';

  systemd.services.tailscale-local-lan-bypass = {
    description = "Prefer local 192.168.8.0/24 over Tailscale policy routing";
    after = [ "tailscaled.service" ];
    wants = [ "tailscaled.service" ];

    serviceConfig = {
      Type = "oneshot";
      ExecStart = applyLocalLanBypass;
    };
  };

  systemd.paths.tailscale-local-lan-bypass = {
    description = "Watch Tailscale state for local-LAN routing changes";
    wantedBy = [ "multi-user.target" ];

    pathConfig = {
      PathChanged = "/var/lib/tailscale/tailscaled.state";
      Unit = "tailscale-local-lan-bypass.service";
    };
  };
}
