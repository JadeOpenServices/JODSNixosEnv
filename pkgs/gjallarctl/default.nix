{
  buildGoModule,
  git,
  lib,
}:
buildGoModule {
  pname = "gjallarctl";
  version = "0.1.0";

  nativeCheckInputs = [ git ];

  src = lib.fileset.toSource {
    root = ../../.;
    fileset = lib.fileset.unions [
      ../../go.mod
      ../../cmd
      ../../pkg
      ../../internal/ai
      ../../internal/hardware
      ../../internal/input
      ../../internal/installercheck
      ../../internal/installer
      ../../internal/preset
      ../../internal/usbtrust
      ../../oddc
    ];
  };

  vendorHash = null;

  subPackages = [
    "cmd/gjallarctl"
    "cmd/gjallar-installer"
    "cmd/gjallar-recovery-maintenance"
    "cmd/oddcctl"
  ];

  env.CGO_ENABLED = "0";
  ldflags = [
    "-s"
    "-w"
  ];

  meta = {
    description = "Safe GjallarOS maintenance and validation tool";
    mainProgram = "gjallarctl";
    license = lib.licenses.mit;
    platforms = lib.platforms.linux;
  };
}
