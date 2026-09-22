{
  lib,
  pkgs,
  settings,
  ...
}:

let
  printing = settings.printingEnable or false;
  networkPrinting = printing && (settings.networkPrintingEnable or false);
in
lib.mkIf printing {

  services.printing = {
    enable = true;

    startWhenNeeded = true;

    listenAddresses = [ "localhost:631" ];
    allowFrom = [ "localhost" ];
    openFirewall = false;

    defaultShared = false;
    browsing = false;
    webInterface = false;

    browsed.enable = false;

    drivers = with pkgs; [
      gutenprint
    ];
  };

  services.ipp-usb.enable = true;

  hardware.sane.enable = true;

  services.avahi = lib.mkIf networkPrinting {
    enable = true;
    nssmdns4 = true;

    openFirewall = true;
  };
}
