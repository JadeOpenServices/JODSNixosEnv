{ pkgs, ... }:
{
  # Vendor-neutral FIDO2 security key tooling. fido2-token (libfido2) shows
  # device info, sets the PIN and resets keys; fido2-manage lists and deletes
  # the passkeys stored on them. systemd's 60-fido-id.rules already grants the
  # logged-in user access, and no PAM login is wired to these keys here.
  environment.systemPackages = with pkgs; [
    libfido2
    fido2-manage
  ];
}
