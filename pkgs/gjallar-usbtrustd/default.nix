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
      ../../cmd/gjallar-usbtrustd
      ../../internal/usbtrust
      ../../oddc
    ];
  };

  vendorHash = null;
  # ODDC, the only dependency, resolves through the go.mod replace to the
  # oddc/ subtree; module mode instead of -mod=vendor needs no vendor dir.
  proxyVendor = true;

  subPackages = [
    "cmd/gjallar-usbtrustd"
  ];

  postInstall = ''
    mkdir -p "$out/share/gjallarOS"
    cp -R "$src/oddc" "$out/share/gjallarOS/oddc"
  '';

  meta = {
    description = "GjallarOS privileged USB trust broker";
    license = lib.licenses.mit;
    mainProgram = "gjallar-usbtrustd";
    platforms = lib.platforms.linux;
  };
}
