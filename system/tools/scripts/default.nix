{ pkgs, lib, settings, ... }:
let
  gjallarctl = pkgs.callPackage ../../../pkgs/gjallarctl { };
  checkInstaller = pkgs.writeShellScriptBin "check-installer" ''
    exec ${gjallarctl}/bin/gjallarctl check --repo ${lib.escapeShellArg settings.dotfilesDir} "$@"
  '';

  replaceSettings =
    file:
    builtins.replaceStrings
      [ "__REPO_ROOT__" "__HOSTNAME__" ]
      [ settings.dotfilesDir settings.hostname ]
      (builtins.readFile file);
in
{
  environment.systemPackages = [
    (pkgs.writeShellScriptBin "helpme" (builtins.readFile ./help.sh))
    (pkgs.writeShellScriptBin "rebuild" (replaceSettings ./rebuild.sh))
    (pkgs.writeShellScriptBin "update" (replaceSettings ./update.sh))
    (pkgs.writeShellScriptBin "cleanup" (builtins.readFile ./cleanup.sh))
    (pkgs.writeShellScriptBin "cleanup-old-generations" (
      builtins.readFile ../functions/cleanup-old-generations.sh
    ))
    (pkgs.writeShellScriptBin "thermal-status" (builtins.readFile ./thermal-status.sh))
    (pkgs.writeShellScriptBin "thermal-test" (builtins.readFile ./thermal-test.sh))
    gjallarctl
    checkInstaller
    pkgs.yad
  ];
}
