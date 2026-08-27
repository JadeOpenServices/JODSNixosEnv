#!/usr/bin/env bash
set -euo pipefail
[[ $# -eq 2 ]] || { printf 'Usage: thermal-test <off|on|status> <process-pattern>\n' >&2; exit 2; }
action="$1"; pattern="$2"; pids="$(pgrep -f "$pattern" || true)"
[[ -n "$pids" ]] || { printf 'No matching processes.\n'; exit 0; }
case "$action" in
    off) xargs -r kill -STOP <<<"$pids" ;;
    on) xargs -r kill -CONT <<<"$pids" ;;
    status) ps -o pid,stat,comm,args= -p "$(tr '\n' ',' <<<"$pids" | sed 's/,$//')" ;;
    *) printf 'Unknown action: %s\n' "$action" >&2; exit 2 ;;
esac
