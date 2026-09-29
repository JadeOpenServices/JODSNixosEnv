{
  buildGoModule,
  git,
  lib,
  runtimeShell,
  systemd,
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

  postInstall = ''
    cat > "$out/bin/gjallar-sudo-askpass" <<'ASKPASS'
#!${runtimeShell}
# sudo runs askpass with its own stdin, which gjallarctl detaches. Prompt on
# the controlling terminal; without one, fail instead of waiting forever for
# a password agent that does not exist.
exec ${lib.getExe' systemd "systemd-ask-password"} \
  --echo=no \
  --timeout=0 \
  "$@" </dev/tty
ASKPASS
    chmod 0555 "$out/bin/gjallar-sudo-askpass"

    cat > "$out/bin/gjallar-sudo-auth" <<'AUTH'
#!${runtimeShell}
exit 0
AUTH
    chmod 0555 "$out/bin/gjallar-sudo-auth"
  '';

  meta = {
    description = "Safe GjallarOS maintenance and validation tool";
    mainProgram = "gjallarctl";
    license = lib.licenses.mit;
    platforms = lib.platforms.linux;
  };
}
