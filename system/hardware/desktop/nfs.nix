{pkgs, ...}:
{
    environment.systemPackages = with pkgs; [ nfs-utils ];
    boot.supportedFilesystems = [ "nfs" ];
    services.nfs = {};
    fileSystems."/home/serpentian/Drives/network" = {
        device = "192.0.2.219:/home/serpentian/media";
        fsType = "nfs";
    };
}
