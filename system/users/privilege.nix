{ lib, settings, ... }:
let
  rootPasswordReady = settings.rootPasswordFile != "" && builtins.pathExists settings.rootPasswordFile;
in
{
  # Merely requesting JODS enrollment must not remove the only local
  # administrator. A verified policy may harden privileges after enrollment.
  users.users.root = lib.mkMerge [
    (lib.mkIf rootPasswordReady {
      hashedPasswordFile = settings.rootPasswordFile;
    })
  ];

  nix.settings.trusted-users = lib.mkForce [ "root" ];

  security.polkit.adminIdentities =
    if rootPasswordReady then [ "unix-user:root" ]
    else [ "unix-group:wheel" ];
}
