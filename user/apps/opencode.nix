{
  lib,
  pkgs,
  settings,
  ...
}:

let
  aiEnabled = if settings ? aiEnable then settings.aiEnable else false;

  aiModel = settings.aiModel;

  gjallarAi = pkgs.writeShellScriptBin "gjallar-ai" ''
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
        require systemd-run
        require realpath
        require mktemp
        require rm

        repo="$(git rev-parse --show-toplevel 2>/dev/null || true)"
        [[ -n "$repo" ]] ||
          die "Start gjallar-ai inside a Git repository."

        repo="$(realpath "$repo")"

        session="$(mktemp -d -t gjallar-ai.XXXXXXXX)"
        cleanup() {
          rm -rf -- "$session"
        }
        trap cleanup EXIT INT TERM HUP

        mkdir -p \
          "$session/home" \
          "$session/config" \
          "$session/cache" \
          "$session/data" \
          "$session/state" \
          "$session/tmp"

        config="$session/config/opencode.json"

        cat > "$config" <<EOF
    {
      "\$schema": "https://opencode.ai/config.json",

      "autoupdate": false,
      "share": "disabled",

      "provider": {
        "ollama": {
          "npm": "@ai-sdk/openai-compatible",
          "name": "GjallarOS Ollama",
          "options": {
            "baseURL": "http://127.0.0.1:11434/v1"
          },
          "models": {
            "${aiModel}": {
              "name": "GjallarOS Local ${aiModel}"
            }
          }
        }
      },

      "model": "ollama/${aiModel}",

      "permissions": [
        { "action": "*", "resource": "*", "effect": "deny" },

        { "action": "read", "resource": "*", "effect": "allow" },
        { "action": "glob", "resource": "*", "effect": "allow" },
        { "action": "grep", "resource": "*", "effect": "allow" },
        { "action": "list", "resource": "*", "effect": "allow" },

        { "action": "edit", "resource": "*", "effect": "allow" },

        { "action": "read", "resource": "*.env", "effect": "deny" },
        { "action": "read", "resource": "*.env.*", "effect": "deny" },
        { "action": "read", "resource": "**/.ssh/**", "effect": "deny" },
        { "action": "read", "resource": "**/.gnupg/**", "effect": "deny" },

        { "action": "external_directory", "resource": "*", "effect": "deny" },

        { "action": "webfetch", "resource": "*", "effect": "deny" },
        { "action": "websearch", "resource": "*", "effect": "deny" },

        { "action": "subagent", "resource": "*", "effect": "deny" },

        { "action": "shell", "resource": "*", "effect": "deny" },

        { "action": "shell", "resource": "git status *", "effect": "allow" },
        { "action": "shell", "resource": "git diff *", "effect": "allow" },
        { "action": "shell", "resource": "git log *", "effect": "allow" },
        { "action": "shell", "resource": "git branch *", "effect": "allow" },
        { "action": "shell", "resource": "git show *", "effect": "allow" },

        { "action": "shell", "resource": "nix flake check *", "effect": "allow" },
        { "action": "shell", "resource": "nix build *", "effect": "allow" },
        { "action": "shell", "resource": "nix eval *", "effect": "allow" },

        { "action": "shell", "resource": "sudo *", "effect": "deny" },
        { "action": "shell", "resource": "doas *", "effect": "deny" },
        { "action": "shell", "resource": "pkexec *", "effect": "deny" },

        { "action": "shell", "resource": "git push *", "effect": "deny" },
        { "action": "shell", "resource": "git reset *", "effect": "deny" },
        { "action": "shell", "resource": "git clean *", "effect": "deny" },

        { "action": "shell", "resource": "rm *", "effect": "deny" },
        { "action": "shell", "resource": "rmdir *", "effect": "deny" },
        { "action": "shell", "resource": "mv *", "effect": "deny" },
        { "action": "shell", "resource": "cp *", "effect": "deny" },

        { "action": "shell", "resource": "curl *", "effect": "deny" },
        { "action": "shell", "resource": "wget *", "effect": "deny" },
        { "action": "shell", "resource": "ssh *", "effect": "deny" },
        { "action": "shell", "resource": "scp *", "effect": "deny" },
        { "action": "shell", "resource": "sftp *", "effect": "deny" }
      ],

      "agent": {
        "gjallar": {
          "description": "GjallarOS local engineering agent",
          "mode": "primary",
          "model": "ollama/${aiModel}",
          "steps": 100,

          "permission": {
            "shell": {
              "*": "deny",
              "git status *": "allow",
              "git diff *": "allow",
              "git log *": "allow",
              "git branch *": "allow",
              "git show *": "allow",
              "nix flake check *": "allow",
              "nix build *": "allow",
              "nix eval *": "allow"
            },
            "webfetch": "deny",
            "websearch": "deny",
            "external_directory": "deny",
            "task": "deny"
          }
        }
      }
    }
    EOF

        exec systemd-run \
          --user \
          --wait \
          --pipe \
          --collect \
          --service-type=exec \
          -p "ProtectSystem=strict" \
          -p "ProtectHome=tmpfs" \
          -p "PrivateTmp=yes" \
          -p "NoNewPrivileges=yes" \
          -p "PrivateDevices=yes" \
          -p "PrivateUsers=no" \
          -p "RestrictSUIDSGID=yes" \
          -p "CapabilityBoundingSet=" \
          -p "LockPersonality=yes" \
          -p "ProtectKernelTunables=yes" \
          -p "ProtectKernelModules=yes" \
          -p "ProtectKernelLogs=yes" \
          -p "ProtectControlGroups=yes" \
          -p "RestrictNamespaces=yes" \
          -p "RestrictRealtime=yes" \
          -p "IPAddressDeny=any" \
          -p "IPAddressAllow=localhost" \

          -p "RestrictAddressFamilies=AF_UNIX AF_INET AF_INET6" \
          -p "BindPaths=$repo" \
          -p "BindReadOnlyPaths=$session/config" \
          -p "BindReadOnlyPaths=$session/cache" \
          -p "BindReadOnlyPaths=$session/data" \
          -p "BindReadOnlyPaths=$session/state" \
          -p "ReadWritePaths=$repo" \
          -p "ReadWritePaths=$session" \
          -p "Environment=HOME=$session/home" \
          -p "Environment=XDG_CONFIG_HOME=$session/config" \
          -p "Environment=XDG_CACHE_HOME=$session/cache" \
          -p "Environment=XDG_DATA_HOME=$session/data" \
          -p "Environment=XDG_STATE_HOME=$session/state" \
          -p "Environment=OPENCODE_CONFIG=$config" \
          -p "Environment=OPENAI_API_KEY=ollama" \
          -p "Environment=OPENAI_BASE_URL=http://127.0.0.1:11434/v1" \
          -p "Environment=ANTHROPIC_API_KEY=" \
          -p "Environment=ANTHROPIC_AUTH_TOKEN=" \
          -p "Environment=GOOGLE_API_KEY=" \
          -p "Environment=OPENROUTER_API_KEY=" \
          -- \
          ${pkgs.opencode}/bin/opencode \
          --cwd "$repo" \
          "$@"
  '';

in
lib.mkIf aiEnabled {
  programs.opencode = {
    enable = true;
  };

  home.packages = [
    gjallarAi

    (pkgs.writeShellScriptBin "opencode-local" ''
      set -euo pipefail
      export OPENAI_API_KEY=ollama
      export OPENAI_BASE_URL=http://127.0.0.1:11434/v1
      unset ANTHROPIC_API_KEY ANTHROPIC_AUTH_TOKEN GOOGLE_API_KEY OPENROUTER_API_KEY
      exec ${pkgs.opencode}/bin/opencode "$@"
    '')

    (pkgs.writeShellScriptBin "gjallar-research" ''
      set -euo pipefail

      url="''${1:-}"
      [[ "$url" =~ ^https:// ]] || {
        echo "Usage: gjallar-research https://approved-host/path" >&2
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
  ];
}
