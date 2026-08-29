{ pkgs, settings, ... }:
let
    replaceSettings = file: builtins.replaceStrings
        [ "__REPO_ROOT__" "__HOSTNAME__" ]
        [ settings.dotfilesDir settings.hostname ]
        (builtins.readFile file);
in {
    environment.systemPackages = [
        (pkgs.writeShellScriptBin "helpme" (builtins.readFile ./help.sh))
        (pkgs.writeShellScriptBin "rebuild" (replaceSettings ./rebuild.sh))
        (pkgs.writeShellScriptBin "update" (replaceSettings ./update.sh))
        (pkgs.writeShellScriptBin "cleanup" (builtins.readFile ./cleanup.sh))
        (pkgs.writeShellScriptBin "thermal-status" (builtins.readFile ./thermal-status.sh))
        (pkgs.writeShellScriptBin "thermal-test" (builtins.readFile ./thermal-test.sh))
        pkgs.yad
    ];
}
