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
  env.CGO_ENABLED = "0";
  ldflags = [ "-s" "-w" ];

  meta = {
    description = "Safe GjallarOS maintenance and validation tool";
    mainProgram = "gjallarctl";
    license = lib.licenses.mit;
    platforms = lib.platforms.linux;
  };
}
