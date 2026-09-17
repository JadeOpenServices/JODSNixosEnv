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
  # Printing/scanning is explicit human intent.
  # ODDC may describe hardware capability, but never enables this feature.

  services.printing = {
    enable = true;

    # cupsd is socket-activated instead of permanently running.
    startWhenNeeded = true;

    # CUPS itself is never exposed to the network.
    listenAddresses = [ "localhost:631" ];
    allowFrom = [ "localhost" ];
    openFirewall = false;

    # This workstation does not share printers.
    defaultShared = false;
    browsing = false;
    webInterface = false;

    # Do not continuously import remote queues.
    browsed.enable = false;

    drivers = with pkgs; [
      gutenprint
    ];
  };

  # Local driverless USB printing.
  services.ipp-usb.enable = true;

  # Local scanner backends/device access.
  hardware.sane.enable = true;

  # Network printer discovery is a separate explicit installer intent.
  services.avahi = lib.mkIf networkPrinting {
    enable = true;
    nssmdns4 = true;

    # Only network-printer discovery owns this exposure.
    openFirewall = true;
  };
}
