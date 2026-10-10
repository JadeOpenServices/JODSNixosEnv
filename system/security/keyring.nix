{ ... }:
{
  services.gnome.gnome-keyring.enable = true;

  security.pam.services = {
    greetd.enableGnomeKeyring = true;
    login.enableGnomeKeyring = true;
    # passwd knows the old password, so the login keyring follows the new
    # one; otherwise the next login leaves it locked and an app asks for
    # the old password. A root reset cannot re-encrypt it.
    passwd.enableGnomeKeyring = true;
  };
}
