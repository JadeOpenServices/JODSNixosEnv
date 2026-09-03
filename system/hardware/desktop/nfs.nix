{ pkgs, ... }:
{
  environment.systemPackages = with pkgs; [ nfs-utils ];
  boot.supportedFilesystems = [ "nfs" ];
  services.nfs = { };
  # Add a machine-specific NFS mount here when needed. The original mount
  # pointed to the author's private network and is intentionally disabled.
}
