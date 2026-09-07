{ lib, settings, ... }:
let
  rootPasswordReady =
    settings.rootPasswordFile != "";
in
{
  # Merely requesting JODS enrollment must not remove the only local
  # administrator. A verified policy may harden privileges after enrollment.
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

  # Administrative authority follows actual endpoint-management state.
  #
  # Merely having a root password configured must never remove the normal
  # local administrator from wheel. JODS/MDM-managed endpoints deliberately
  # move administrative authorization to root.
  security.polkit.adminIdentities =
    if settings.endpointManagedDevice then
      [ "unix-user:root" ]
    else
      [ "unix-group:wheel" ];
}
