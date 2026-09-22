{ lib, settings, ... }:
let
  rootPasswordReady =
    settings.rootPasswordFile != "";
in
{
  users.users.root = lib.mkMerge [
    (lib.mkIf rootPasswordReady {
      hashedPasswordFile = settings.rootPasswordFile;
    })
  ];

  assertions = [
    {
      assertion =
        !settings.endpointManagedDevice
        || rootPasswordReady;

      message =
        "endpointManagedDevice=true requires a configured and existing "
        + "rootPasswordFile before local wheel/Polkit administration "
        + "may be revoked.";
    }
  ];

  nix.settings.trusted-users = lib.mkForce [ "root" ];

  security.polkit.adminIdentities =
    if settings.endpointManagedDevice then
      [ "unix-user:root" ]
    else
      [ "unix-group:wheel" ];
}
