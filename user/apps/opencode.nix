{
  lib,
  pkgs,
  settings,
  ...
}:

let
  aiEnabled = if settings ? aiEnable then settings.aiEnable else false;

  aiModel = "gjallaros-caveman-ai";
  agentMode = if settings ? aiAgentMode then settings.aiAgentMode else "workspace";
  ownerMode = agentMode != "workspace";
  gjallarctl = pkgs.callPackage ../../pkgs/gjallarctl { };

  gjallarAi = pkgs.writeShellScriptBin "gjallar-ai" ''
    set -euo pipefail

    die() {
      printf 'gjallar-ai: %s\n' "$*" >&2
      exit 1
    }

    for command in \
      git \
      realpath \
      systemctl \
      systemd-escape \
      socat
    do
      command -v "$command" >/dev/null 2>&1 ||
        die "required command not found: $command"
    done

    repo="$(
      git rev-parse \
        --show-toplevel \
        2>/dev/null ||
        true
    )"

    [ -n "$repo" ] ||
      die "Start gjallar-ai inside a Git repository."

    repo="$(realpath "$repo")"

    escaped="$(
      systemd-escape \
        --path \
        "$repo"
    )"

    [ -n "$escaped" ] ||
      die "failed to encode workspace"

    unit="gjallar-ai-session@$escaped.service"

    socket="/run/gjallar-ai-session-$escaped/console.sock"

    cleanup() {
      systemctl stop "$unit" >/dev/null 2>&1 || true
    }

    trap cleanup EXIT INT TERM HUP

    systemctl start "$unit" ||
      die "failed to start controlled AI system session"

    ready=false

    for _ in $(${pkgs.coreutils}/bin/seq 1 100); do
      if [ -S "$socket" ]; then
        ready=true
        break
      fi

      state="$(
        systemctl show \
          "$unit" \
          --property=ActiveState \
          --value \
          2>/dev/null ||
          true
      )"

      case "$state" in
        failed|inactive)
          systemctl status \
            "$unit" \
            --no-pager \
            >&2 ||
            true

          die \
            "controlled AI session exited before console startup"
          ;;
      esac

      ${pkgs.coreutils}/bin/sleep 0.05
    done

    if [ "$ready" != true ]; then
      systemctl status \
        "$unit" \
        --no-pager \
        >&2 ||
        true

      die "controlled AI console did not become ready"
    fi

    set +e

    ${pkgs.socat}/bin/socat \
      STDIO,raw,echo=0 \
      UNIX-CONNECT:"$socket"

    rc=$?

    set -e

    exit "$rc"
  '';


  gjallarAiDesktop = pkgs.makeDesktopItem {
    name = "gjallarOS-ai";
    desktopName = "gjallarOS AI";
    comment = "Secure local GjallarOS engineering agent";
    exec = "${gjallarAi}/bin/gjallar-ai";
    icon = "${pkgs.papirus-icon-theme}/share/icons/Papirus/64x64/apps/devassistant.svg";
    terminal = true;
    categories = [ "Development" ];
  };

in
lib.mkIf aiEnabled {
  programs.opencode = {
    enable = true;
  };

  home.packages = [
    gjallarAi
    gjallarAiDesktop

    (pkgs.writeShellScriptBin "gjallar-agent-tool" ''
      set -euo pipefail
      workspace="$(git rev-parse --show-toplevel 2>/dev/null)" || {
        echo "gjallar-agent-tool: start inside a Git workspace" >&2; exit 2;
      }
      exec ${gjallarctl}/bin/gjallarctl ai tool \
        --workspace "$workspace" \
        --mode ${lib.escapeShellArg agentMode} \
        --approval "''${XDG_RUNTIME_DIR}/gjallar-ai/approval.json" \
        --audit "''${XDG_STATE_HOME:-$HOME/.local/state}/gjallar-ai/audit.jsonl" \
        "$@"
    '')

    (pkgs.writeShellScriptBin "opencode-local" ''
      set -euo pipefail

      unset OPENAI_BASE_URL OPENAI_API_KEY

      unset         ANTHROPIC_API_KEY         ANTHROPIC_AUTH_TOKEN         GOOGLE_API_KEY         OPENROUTER_API_KEY

      exec ${gjallarAi}/bin/gjallar-ai "$@"
    '')

    (pkgs.writeShellScriptBin "gjallar-ai-approve" ''
      set -euo pipefail
      workspace="$(git rev-parse --show-toplevel 2>/dev/null)" || {
        echo "gjallar-ai-approve: start inside the target Git workspace" >&2; exit 2;
      }
      runtime="''${XDG_RUNTIME_DIR:-/run/user/$(id -u)}"
      exec ${gjallarctl}/bin/gjallarctl ai approve \
        --workspace "$workspace" --grant "$runtime/gjallar-ai/approval.json" "$@"
    '')

    (pkgs.writeShellScriptBin "gjallar-research" ''
      set -euo pipefail

      url="''${1:-}"
      [[ "$url" =~ ^https:// ]] || {
        echo "Usage: gjallar-research https://approved-host/path" >&2
        exit 2
      }

      exec ${pkgs.curl}/bin/curl \
        --fail \
        --silent \
        --show-error \
        --unix-socket /run/gjallar-ai/research.sock \
        --max-time 15 \
        --get --data-urlencode "url=$url" \
        http://localhost/research
    '')
  ];
}
