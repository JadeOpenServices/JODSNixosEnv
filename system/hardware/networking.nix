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
}
