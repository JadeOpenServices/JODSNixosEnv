{ lib, pkgs, settings, ...}:
lib.mkIf (if settings ? aiEnable then settings.aiEnable else false) {
    programs.opencode = {
        enable = true;
        # package = inputs.opencode.packages.${pkgs.system}.opencode;
    };

    # Keep cloud credentials out of the default environment. Cloud tools can
    # be added explicitly per user when needed.
    home.packages = [
      (pkgs.writeShellScriptBin "opencode-local" ''
      set -euo pipefail
      export OPENAI_API_KEY=ollama
      export OPENAI_BASE_URL=http://127.0.0.1:11434/v1
      unset ANTHROPIC_API_KEY ANTHROPIC_AUTH_TOKEN GOOGLE_API_KEY OPENROUTER_API_KEY
      exec opencode "$@"
      '')
      (pkgs.writeShellScriptBin "gjallar-research" ''
        set -euo pipefail
        url="''${1:-}"
        [[ "$url" =~ ^https:// ]] || { echo 'Usage: gjallar-research https://approved-host/path' >&2; exit 2; }
        host="''${url#https://}"; host="''${host%%/*}"
        case "$host" in
          nixos.org|*.nixos.org|github.com|*.github.com|docs.python.org|developer.mozilla.org) ;;
          *) echo "Blocked host: $host (not on the GjallarOS research allowlist)" >&2; exit 1 ;;
        esac
        exec ${pkgs.curl}/bin/curl --fail --silent --show-error --location \
          --proto '=https' --max-time 15 --connect-timeout 5 --max-filesize 2097152 "$url"
      '')
    ];
}
