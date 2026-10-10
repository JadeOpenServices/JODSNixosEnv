{
  pkgs,
  gjallarctlPackage,
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
      gjallarctlPackage
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
    (pkgs.writeTextDir "share/zsh/site-functions/_gjallar-secure-boot" ''
      #compdef gjallar-secure-boot
      local -a cmds=(
        'status:boot, key and signed artifact state'
        'enroll:enroll the GjallarOS Secure Boot keys'
        'firmware:list firmware devices'
      )
      _describe -t commands 'gjallar-secure-boot command' cmds
    '')
  ];
}
