{ ... }:
{
  # Provide Secret Service and unlock it during password login.
  services.gnome.gnome-keyring.enable = true;

  security.pam.services = {
    greetd.enableGnomeKeyring = true;
    login.enableGnomeKeyring = true;
  };
}
