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
      ../../go.sum
      ../../cmd
      ../../internal/ai
      ../../internal/hardware
      ../../internal/input
      ../../internal/installercheck
      ../../internal/installer
      ../../internal/nettrust
      ../../internal/oddccli
      ../../internal/preset
      ../../internal/usbtrust
    ];
  };

  vendorHash = "sha256-8HOS0S6am4g59tpFI7R5FOCCmm3ptHwgsLlZu90IxpU=";
  # Module mode: ODDC's catalog files stay in the module cache, which the
  # tests read as the real catalog.
  proxyVendor = true;

  subPackages = [
    "cmd/gjallarctl"
    "cmd/gjallar-installer"
    "cmd/gjallar-recovery-maintenance"
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
