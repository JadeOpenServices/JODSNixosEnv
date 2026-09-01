#!/usr/bin/env bash

set -euo pipefail

repo="${GJALLAROS_REPO:-__REPO_ROOT__}"

source "$repo/system/tools/functions/cleanup-old-generations.sh"

host="${GJALLAROS_HOST:-__HOSTNAME__}"

messages_json="$repo/system/tools/scripts/rebuild-messages.json"

debug=0
cleanup_enabled=1
args=()

for arg in "$@"; do
    case "$arg" in
        -d|--debug)
            debug=1
            ;;
        -n|--no-cleanup)
            cleanup_enabled=0
            ;;
        *)
            args+=("$arg")
            ;;
    esac
done

messages=()

while IFS= read -r line; do
    if [[ "$line" == *'"Rebuilding NixOS for $runtime_host'* ]]; then
        message="${line#*\"}"
        message="${message%\",*}"
        message="${message%\"}"
        message="${message//\$runtime_host/$host}"
        messages+=("$message")
    fi
done < "$messages_json"

if ((${#messages[@]} == 0)); then
    printf '[GjallarOS] Error: no rebuild messages found in %s\n' "$messages_json" >&2
    exit 1
fi

tmp_log="$(mktemp)"
ui_pid=""
start_time=$SECONDS

cleanup() {
    if [[ -n "$ui_pid" ]]; then
        kill "$ui_pid" 2>/dev/null || true
        wait "$ui_pid" 2>/dev/null || true
    fi

    rm -f "$tmp_log"
    printf '\r\033[2K'
}

cancel() {
    trap - INT TERM

    if [[ -n "$ui_pid" ]]; then
        kill "$ui_pid" 2>/dev/null || true
        wait "$ui_pid" 2>/dev/null || true
    fi

    elapsed=$((SECONDS - start_time))
    minutes=$((elapsed / 60))
    seconds=$((elapsed % 60))

    printf '\r\033[2K'
    printf '⏱ %02d:%02d  ✗ Rebuild cancelled.\n' "$minutes" "$seconds"

    rm -f "$tmp_log"
    exit 130
}

trap cleanup EXIT
trap cancel INT TERM

sudo -v

if ((debug)); then
    printf '[GjallarOS] Debug rebuild for %s\n' "$host"
    printf '[GjallarOS] Flake: %s#%s\n\n' "$repo" "$host"

    sudo nixos-rebuild switch \
        --flake "$repo#$host" \
        --show-trace \
        "${args[@]}"

    status=$?

    if ((status == 0 && cleanup_enabled)); then
        cleanup_old_generations
    fi

    elapsed=$((SECONDS - start_time))
    minutes=$((elapsed / 60))
    seconds=$((elapsed % 60))

    if ((status == 0)); then
        printf '\n⏱ %02d:%02d  ✓ Rebuild completed successfully.\n' \
            "$minutes" "$seconds"
    else
        printf '\n⏱ %02d:%02d  ✗ Rebuild failed.\n' \
            "$minutes" "$seconds"
    fi

    exit "$status"
fi

(
    index=$((RANDOM % ${#messages[@]}))
    last_index=$index
    next_change=5

    while true; do
        elapsed=$((SECONDS - start_time))
        minutes=$((elapsed / 60))
        seconds=$((elapsed % 60))

        line="⏱ $(printf '%02d:%02d' "$minutes" "$seconds")  ${messages[index]}"

        printf '\r\033[2K%s' "$line"

        if ((elapsed >= next_change)); then
            if ((${#messages[@]} > 1)); then
                while :; do
                    index=$((RANDOM % ${#messages[@]}))
                    ((index != last_index)) && break
                done
            fi

            last_index=$index
            next_change=$((next_change + 5))
        fi

        sleep 1
    done
) &

ui_pid=$!

set +e

sudo nixos-rebuild switch \
    --flake "$repo#$host" \
    "${args[@]}" >"$tmp_log" 2>&1

status=$?

set -e

if ((status == 0 && cleanup_enabled)); then
    cleanup_old_generations
fi

elapsed=$((SECONDS - start_time))
minutes=$((elapsed / 60))
seconds=$((elapsed % 60))

kill "$ui_pid" 2>/dev/null || true
wait "$ui_pid" 2>/dev/null || true
ui_pid=""

printf '\r\033[2K'

if ((status == 0)); then
    printf '⏱ %02d:%02d  ✓ Rebuild completed successfully.\n' \
        "$minutes" "$seconds"
else
    printf '⏱ %02d:%02d  ✗ Rebuild failed.\n' \
        "$minutes" "$seconds"

    printf '\n'
    cat "$tmp_log" >&2

    exit "$status"
fi
