{ lib, settings, ... }:
let
  rootPasswordReady = settings.rootPasswordFile != "" && builtins.pathExists settings.rootPasswordFile;
in
{
  # System activation is never delegated to desktop users. Unmanaged owners
  # can become root with the separately configured root password; JODS-managed
  # endpoints keep interactive root authentication locked.
  users.users.root = lib.mkMerge [
    (lib.mkIf settings.endpointManagedDevice { hashedPassword = "!"; })
    (lib.mkIf (!settings.endpointManagedDevice && rootPasswordReady) {
      hashedPasswordFile = settings.rootPasswordFile;
    })
  ];

  nix.settings.trusted-users = lib.mkForce [ "root" ];

  security.polkit.adminIdentities =
    if settings.endpointManagedDevice then [ ]
    else if rootPasswordReady then [ "unix-user:root" ]
    else [ "unix-group:wheel" ];
}
