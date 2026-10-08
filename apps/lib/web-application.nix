{ lib, pkgs }:
let
  urlLib = import ./url.nix { inherit lib; };
in
{
  mkIsolatedWebApplication =
    {
      name,
      browser,
      profile,
      url,
      browserArguments ? [ ],
      # https hosts the launcher may open instead of url; empty = url only.
      runtimeHosts ? [ ],
    }:
    pkgs.writeShellScriptBin name ''
      set -euo pipefail
      profile_directory="$HOME/${profile}"
      application_url=${lib.escapeShellArg url}

      ${lib.optionalString (runtimeHosts != [ ]) ''
        if [ "$#" -gt 0 ]; then
          url="$1"
          ${urlLib.parseUrl}
          case "$url_scheme:$url_host" in
            ${urlLib.hostPattern "https" runtimeHosts}) application_url="$url" ;;
            *)
              echo "${name}: refusing URL outside its hosts" >&2
              exit 2
              ;;
          esac
        fi
      ''}

      exec ${lib.getExe browser} \
        ${lib.escapeShellArgs browserArguments} \
        --user-data-dir="$profile_directory" \
        --app="$application_url"
    '';
}
