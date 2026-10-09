{ pkgs, lib, settings, ... }:
let
  gjallarctl = pkgs.callPackage ../../../pkgs/gjallarctl { };
  checkInstaller = pkgs.writeShellScriptBin "check-installer" ''
    exec ${gjallarctl}/bin/gjallarctl check --repo ${lib.escapeShellArg settings.dotfilesDir} "$@"
  '';
  preflight = pkgs.writeShellScriptBin "gjallar-preflight" ''
    exec ${gjallarctl}/bin/gjallarctl preflight --repo ${lib.escapeShellArg settings.dotfilesDir} "$@"
  '';
  # Rebuild with the checkout's gjallarctl: an installed binary older than
  # the checkout rejects config keys it does not know yet ("unknown field
  # apps", 2026-10-08) and could never deploy its own successor.
  rebuild = pkgs.writeShellScriptBin "rebuild" ''
    repo=${lib.escapeShellArg settings.dotfilesDir}
    if [ -n "$repo" ] && [ -f "$repo/pkgs/gjallarctl/default.nix" ]; then
      echo "[GjallarOS] Building gjallarctl from $repo..." >&2
      errors="$(${pkgs.coreutils}/bin/mktemp)"
      if current="$(${pkgs.nix}/bin/nix-build --no-out-link -E \
        "(import ${pkgs.path} { }).callPackage (/. + \"$repo/pkgs/gjallarctl\") { }" 2>"$errors")"; then
        rm -f "$errors"
        exec "$current/bin/gjallarctl" rebuild "$@"
      fi
      echo "[GjallarOS] Building it failed; using the installed gjallarctl:" >&2
      ${pkgs.coreutils}/bin/tail -n 3 "$errors" >&2
      rm -f "$errors"
    fi
    exec ${gjallarctl}/bin/gjallarctl rebuild "$@"
  '';
  goCommand = name: command: pkgs.writeShellScriptBin name ''
    exec ${gjallarctl}/bin/gjallarctl ${command} "$@"
  '';

in
{
  # One fixed place on every deployment that names the system's checkout;
  # gjallarctl falls back to it when no --repo is given.
  environment.etc."gjallar/repository" = lib.mkIf (settings.dotfilesDir != "") {
    text = settings.dotfilesDir + "\n";
  };

  environment.systemPackages = [
    (goCommand "helpme" "helpme")
    rebuild
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
