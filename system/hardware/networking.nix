{ lib, pkgs, settings, ... }:
let
  jodsHostsFile = "${settings.dotfilesDir}/local/jods-hosts";
in
{
  networking.networkmanager.enable = true;
  # Disable Wi-Fi power saving; it can cause severe throughput/latency
  # regressions on laptop chipsets while still allowing suspend normally.
  networking.networkmanager.wifi.powersave = false;
  # Flakes omit ignored files from their source. Merge this machine-local file
  # at activation time, after Nix has generated its normal hosts file.
  system.activationScripts.jodsHosts = {
    deps = [ "etc" ];
    text = ''
      local_hosts=${lib.escapeShellArg jodsHostsFile}
      if [ -f "$local_hosts" ]; then
        ${pkgs.gawk}/bin/awk '
        /^[[:space:]]*($|#)/ { next }
        NF != 2 || $2 != "admin.oss-ad.eu" {
          print "ERROR: local/jods-hosts must contain only: IP admin.oss-ad.eu" > "/dev/stderr"
          failed=1
        }
        END { exit failed }
        ' "$local_hosts"

        generated="$(${pkgs.coreutils}/bin/readlink -f /etc/hosts)"
        tmp="$(${pkgs.coreutils}/bin/mktemp /etc/.hosts-jods.XXXXXX)"
        ${pkgs.coreutils}/bin/cat "$generated" "$local_hosts" > "$tmp"
        ${pkgs.coreutils}/bin/chmod 0644 "$tmp"
        ${pkgs.coreutils}/bin/mv -fT "$tmp" /etc/hosts
      fi
    '';
  };
  hardware.enableRedistributableFirmware = true;
  hardware.wirelessRegulatoryDatabase = true;


  # gjallarOS NetworkManager fast-start BEGIN
  #
  # Portable systems often have unused Ethernet adapters with no carrier.
  # NetworkManager can otherwise wait several seconds on those devices before
  # declaring startup complete. Keep the timeout short while leaving Wi-Fi
  # association/DHCP behavior untouched.
  networking.networkmanager.settings."device-gjallar-ethernet-fast-start" = {
    match-device = "type:ethernet";
    carrier-wait-timeout = 1000;
  };

  # Keep network-online useful, but never allow it to hold startup for a full
  # minute. Applications which can react dynamically to connectivity still do.
  systemd.services.NetworkManager-wait-online.environment.NM_ONLINE_TIMEOUT = "12";
  # gjallarOS NetworkManager fast-start END
}
