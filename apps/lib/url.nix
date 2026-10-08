{ lib }:

{
  # Bash: sets url_scheme and url_host (lowercase) from "$url", or fails.
  # url_host is empty when the authority has userinfo or a port, so such URLs
  # never match an application route.
  parseUrl = ''
    case "$url" in
      *[[:space:][:cntrl:]]*)
        echo "rejected URL with whitespace or control characters" >&2
        exit 2
        ;;
      *://*) ;;
      *)
        echo "rejected URL without scheme://" >&2
        exit 2
        ;;
    esac
    url_scheme="''${url%%://*}"
    url_scheme="''${url_scheme,,}"
    url_rest="''${url#*://}"
    url_host="''${url_rest%%[/?#]*}"
    url_host="''${url_host,,}"
    case "$url_host" in
      *@* | *:* | *\\* ) url_host="" ;;
    esac
  '';

  # Bash case pattern matching "<scheme>:<host>" for any of hosts exactly.
  hostPattern =
    scheme: hosts: lib.concatMapStringsSep " | " (host: "${scheme}:${lib.escapeShellArg host}") hosts;
}
