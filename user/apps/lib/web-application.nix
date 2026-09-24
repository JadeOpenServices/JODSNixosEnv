{ lib, pkgs }:

{
  mkIsolatedWebApplication =
    {
      name,
      browser,
      profile,
      url,
      browserArguments ? [ ],
      allowRuntimeUrl ? false,
    }:
    pkgs.writeShellScriptBin name ''
      profile_directory="$HOME/${profile}"
      application_url=${lib.escapeShellArg url}

      ${lib.optionalString allowRuntimeUrl ''
        if [ "$#" -gt 0 ]; then
          application_url="$1"
        fi
      ''}

      exec ${lib.getExe browser} \
        ${lib.escapeShellArgs browserArguments} \
        --user-data-dir="$profile_directory" \
        --app="$application_url"
    '';
}
