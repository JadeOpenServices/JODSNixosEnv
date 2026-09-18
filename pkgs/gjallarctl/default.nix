{
  buildGoModule,
  git,
  lib,
}:
buildGoModule {
  pname = "gjallarctl";
  version = "0.1.0";

  nativeCheckInputs = [ git ];

  # Only the Go module enters the store.  In particular, user.config.json,
  # generated settings, and encrypted secrets never become build inputs.
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
      ../../oddc
    ];
  };

  vendorHash = null;

  # This package owns the established GjallarOS command suite.
  # Privileged feature daemons such as gjallar-usbtrustd are
  # packaged independently by their owning subsystem.
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
