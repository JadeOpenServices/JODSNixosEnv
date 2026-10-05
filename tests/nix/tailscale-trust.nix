{
  nixpkgs,
  system,
  tailscaleModule,
}:
let
  pkgs = nixpkgs.legacyPackages.${system};
  evaluate =
    settings:
    (nixpkgs.lib.nixosSystem {
      inherit system;
      modules = [
        tailscaleModule
        {
          system.stateVersion = "26.05";
          boot.loader.grub.enable = false;
          networking.networkmanager.enable = true;
          fileSystems."/" = {
            device = "/dev/null";
            fsType = "ext4";
          };
        }
      ];
      specialArgs = { inherit settings; };
    }).config;
  policyOf =
    cfg:
    let
      exec = cfg.systemd.services.gjallar-vpn-trust.serviceConfig.ExecStart;
      file = builtins.elemAt (builtins.split "--policy " exec) 2;
    in
    builtins.fromJSON (builtins.readFile file);

  # Generated state from before the installer asked: the old live behaviour.
  legacy = evaluate { };
  legacyNull = evaluate { tailscale = null; };
  vpn = evaluate {
    tailscale = {
      enable = true;
      homeSubnets = [ "192.168.8.0/24" ];
      trustedWifis = [
        "bakasifu-5Ghz"
        "fizzlipuzzli"
      ];
      exitNode = "home-router";
      siteRouterTrust = true;
      siteRouterTargets = [ "192.168.8.1:53" ];
    };
  };
  off = evaluate {
    tailscale = {
      enable = false;
      homeSubnets = [ ];
      trustedWifis = [ ];
      exitNode = "";
      siteRouterTrust = false;
      siteRouterTargets = [ ];
    };
  };
in
assert legacy.services.tailscale.enable;
assert
  policyOf legacy == {
    homeSubnets = [ "192.168.8.0/24" ];
    trustedWifis = [ ];
    exitNode = "";
    siteRouterTrust = false;
    siteRouterTargets = [ ];
  };
assert policyOf legacyNull == policyOf legacy;
assert legacy.services.tailscale.useRoutingFeatures == "none";
assert !(legacy.systemd.services ? tailscale-local-lan-bypass);
assert
  (policyOf vpn).trustedWifis == [
    "bakasifu-5Ghz"
    "fizzlipuzzli"
  ];
assert (policyOf vpn).exitNode == "home-router";
assert vpn.services.tailscale.useRoutingFeatures == "client";
assert vpn.systemd.paths.gjallar-vpn-trust.pathConfig.Unit == "gjallar-vpn-trust.service";
assert builtins.elem "timers.target" vpn.systemd.timers.gjallar-vpn-trust.wantedBy;
assert nixpkgs.lib.hasInfix "restart --no-block gjallar-vpn-trust.service"
  vpn.systemd.services.tailscaled.postStart;
assert !off.services.tailscale.enable;
assert !(off.systemd.services ? gjallar-vpn-trust);
pkgs.runCommand "gjallar-tailscale-trust-check" { } "touch $out"
