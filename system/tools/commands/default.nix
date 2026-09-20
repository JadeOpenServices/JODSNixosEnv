{ pkgs, lib, settings, ... }:
let
  gjallarctl = pkgs.callPackage ../../../pkgs/gjallarctl { };
  checkInstaller = pkgs.writeShellScriptBin "check-installer" ''
    exec ${gjallarctl}/bin/gjallarctl check --repo ${lib.escapeShellArg settings.dotfilesDir} "$@"
  '';
  preflight = pkgs.writeShellScriptBin "gjallar-preflight" ''
    exec ${gjallarctl}/bin/gjallarctl preflight --repo ${lib.escapeShellArg settings.dotfilesDir} "$@"
  '';
  rebuild = pkgs.writeShellScriptBin "rebuild" ''
    exec ${gjallarctl}/bin/gjallarctl rebuild "$@"
  '';
  goCommand = name: command: pkgs.writeShellScriptBin name ''
    exec ${gjallarctl}/bin/gjallarctl ${command} "$@"
  '';

in
{
  environment.systemPackages = [
    (goCommand "helpme" "helpme")
    rebuild
    (pkgs.writeShellScriptBin "update" ''
      export GJALLAROS_REPO=${lib.escapeShellArg settings.dotfilesDir}
      exec ${gjallarctl}/bin/gjallarctl update "$@"
    '')
    (goCommand "cleanup" "cleanup")
    (goCommand "cleanup-old-generations" "cleanup-old-generations")
    (goCommand "thermal-status" "thermal-status")
    (goCommand "thermal-test" "thermal-test")
    gjallarctl
    checkInstaller
    preflight
    pkgs.yad
  ];
}
