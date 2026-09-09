{
  pkgs,
  gjallarSecureBootArtifactVerifier,
  ...
}:
let
  secureBootTool = pkgs.writeShellApplication {
    name = "gjallar-secure-boot";
    runtimeInputs = [
      pkgs.sbctl
      pkgs.systemd
      pkgs.fwupd
      pkgs.gjallarctl
      gjallarSecureBootArtifactVerifier
    ];

    text = ''
      set -euo pipefail

      case "''${1:-status}" in
        status)
          bootctl status
          printf '\n'
          sudo sbctl status
          printf '\nSigned boot artifacts:\n'
          sudo gjallar-verify-secure-boot-artifacts
          ;;
        enroll)
          sudo gjallarctl installer secure-boot-enroll
          ;;
        firmware)
          fwupdmgr get-devices
          ;;
        *)
          printf '%s\n' \
            'Usage: gjallar-secure-boot {status|enroll|firmware}'
          exit 2
          ;;
      esac
    '';
  };
in
{
  environment.systemPackages = [
    secureBootTool
  ];
}
