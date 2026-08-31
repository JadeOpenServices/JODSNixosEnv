{ settings, ... }:
{
    # Use the bios GRUB 2 boot loader.
    boot.loader.grub.enable = true;
    boot.loader.grub.device = "/dev/nvme0n1";

    boot.loader.timeout = if settings.debugFunctions then 5 else 0;
}