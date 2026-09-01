{ lib, pkgs, settings, ... }:

lib.mkIf (if settings ? aiEnable then settings.aiEnable else false) {
  programs.opencode = {
    enable = true;
  };

  home.packages = [
    (pkgs.writeShellScriptBin "opencode-local" ''
      set -euo pipefail

      export OPENAI_API_KEY=ollama
      export OPENAI_BASE_URL=http://127.0.0.1:11434/v1

      unset \
        ANTHROPIC_API_KEY \
        ANTHROPIC_AUTH_TOKEN \
        GOOGLE_API_KEY \
        OPENROUTER_API_KEY

      exec opencode "$@"
    '')

    (pkgs.writeShellScriptBin "gjallar-research" ''
      set -euo pipefail

      url="''${1:-}"

      [[ "$url" =~ ^https:// ]] || {
        echo 'Usage: gjallar-research https://approved-host/path' >&2
        exit 2
      }

      host="''${url#https://}"
      host="''${host%%/*}"

      case "$host" in
        nixos.org|*.nixos.org|github.com|*.github.com|docs.python.org|developer.mozilla.org)
          ;;
        *)
          echo "Blocked host: $host" >&2
          exit 1
          ;;
      esac

      exec ${pkgs.curl}/bin/curl \
        --fail \
        --silent \
        --show-error \
        --location \
        --proto '=https' \
        --max-time 15 \
        --connect-timeout 5 \
        --max-filesize 2097152 \
        "$url"
    '')

    (pkgs.writeShellScriptBin "gjallar-ai" ''
      set -euo pipefail

      die() {
        echo "gjallar-ai: $*" >&2
        exit 1
      }

      require() {
        command -v "$1" >/dev/null 2>&1 ||
          die "required command not found: $1"
      }

      require git
      require opencode

      repo="$(git rev-parse --show-toplevel 2>/dev/null || true)"
      [[ -n "$repo" ]] || die "Start gjallar-ai inside a Git repository."

      repo="$(realpath "$repo")"

      [[ -d "$repo/.git" || -f "$repo/.git" ]] ||
        die "Current directory is not a Git worktree."

      session="$(mktemp -d -t gjallar-ai.XXXXXXXX)"
      trap 'rm -rf -- "$session"' EXIT INT TERM HUP

      mkdir -p \
        "$session/config" \
        "$session/cache" \
        "$session/data" \
        "$session/state"

      config="$session/config/opencode.json"

      cat > "$config" <<EOF2
{
  "\$schema": "https://opencode.ai/config.json",
  "autoupdate": false,
  "share": "disabled",
  "default_agent": "gjallar",

  "provider": {
    "ollama": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "GjallarOS Ollama",
      "options": {
        "baseURL": "http://127.0.0.1:11434/v1"
      },
      "models": {
        "${settings.aiModel}": {
          "name": "GjallarOS Local ${settings.aiModel}"
        }
      }
    }
  },

  "model": "ollama/${settings.aiModel}",

  "permissions": {
    "read": "allow",
    "glob": "allow",
    "grep": "allow",
    "list": "allow",

    "edit": "allow",

    "webfetch": "deny",
    "websearch": "deny",
    "external_directory": "deny",

    "shell": {
      "git status *": "allow",
      "git diff *": "allow",
      "git log *": "allow",
      "git branch *": "allow",
      "git show *": "allow",

      "nix flake check *": "allow",
      "nix build *": "allow",
      "nix eval *": "allow",

      "sudo *": "deny",
      "doas *": "deny",
      "pkexec *": "deny",
      "git push *": "deny",
      "git reset --hard *": "deny",
      "git clean *": "deny",
      "rm *": "deny",
      "rmdir *": "deny",
      "curl *": "deny",
      "wget *": "deny",
      "scp *": "deny",
      "sftp *": "deny",
      "ssh *": "deny"
    }
  },

  "agents": {
    "gjallar": {
      "description": "GjallarOS local engineering agent",
      "mode": "primary",
      "model": "ollama/${settings.aiModel}",
      "steps": 80,
      "system": "You are the GjallarOS engineering agent. Work on the actual repository. When asked to fix an issue, inspect the actual files and error output first. Reproduce or validate the problem when practical. Identify the root cause, make the smallest correct change, and validate it. If validation fails, continue debugging instead of stopping at the first error. Never invent file contents or claim to have executed a command you did not execute. Work only inside the current repository. Never access credentials, SSH keys, GPG keys, environment secrets, or files outside the repository. Never use network access. Never delete files. Never copy repository data outside the repository. Never push Git changes. Never bypass permissions or sandbox restrictions. Prefer precise edits and Nix validation. Report exactly what changed and what validation was actually performed."
    }
  }
}
EOF2

      exec env \
        HOME="$session" \
        XDG_CONFIG_HOME="$session/config" \
        XDG_CACHE_HOME="$session/cache" \
        XDG_DATA_HOME="$session/data" \
        XDG_STATE_HOME="$session/state" \
        OPENCODE_CONFIG="$config" \
        OPENAI_API_KEY=ollama \
        OPENAI_BASE_URL=http://127.0.0.1:11434/v1 \
        ANTHROPIC_API_KEY= \
        ANTHROPIC_AUTH_TOKEN= \
        GOOGLE_API_KEY= \
        OPENROUTER_API_KEY= \
        opencode --cwd "$repo" "$@"
    '')
  ];
}
