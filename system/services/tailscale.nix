{ pkgs, ... }:

let
  applyLocalLanBypass = pkgs.writeShellScript "tailscale-local-lan-bypass" ''
    ${pkgs.iproute2}/bin/ip rule del \
      to 192.168.8.0/24 \
      priority 2500 \
      2>/dev/null || true

    ${pkgs.iproute2}/bin/ip rule add \
      to 192.168.8.0/24 \
      priority 2500 \
      lookup main
  '';
in
{
  services.tailscale.enable = true;

  # Reapply after every tailscaled start/restart.
  systemd.services.tailscaled.postStart = ''
    ${applyLocalLanBypass}
  '';

  # Reapply after tailscale set/preferences changes.
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
