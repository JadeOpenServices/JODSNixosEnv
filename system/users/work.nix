{
  lib,
  pkgs,
  settings,
  ...
}:
lib.mkIf settings.workUserEnable {
  users.users.${settings.workUsername} = {
    isNormalUser = true;
    description = "${settings.username} work account";
    shell = pkgs.${settings.shell};
    hashedPasswordFile = settings.workUserPasswordFile;
    # Podman is rootless; KVM/Nemu are granted for approved work VMs.
    extraGroups = [
      "audio"
      "video"
      "networkmanager"
      "kvm"
    ];
  };
}
