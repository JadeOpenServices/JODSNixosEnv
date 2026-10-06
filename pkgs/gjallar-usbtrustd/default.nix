{
  lib,
  buildGoModule,
}:

buildGoModule {
  pname = "gjallar-usbtrustd";
  version = "0.1.0";

  src = lib.fileset.toSource {
    root = ../..;
    fileset = lib.fileset.unions [
      ../../go.mod
      ../../go.sum
      ../../cmd/gjallar-usbtrustd
      ../../internal/usbtrust
    ];
  };

  vendorHash = "sha256-8HOS0S6am4g59tpFI7R5FOCCmm3ptHwgsLlZu90IxpU=";
  proxyVendor = true;

  subPackages = [
    "cmd/gjallar-usbtrustd"
  ];

  meta = {
    description = "GjallarOS privileged USB trust broker";
    license = lib.licenses.mit;
    mainProgram = "gjallar-usbtrustd";
    platforms = lib.platforms.linux;
  };
}
