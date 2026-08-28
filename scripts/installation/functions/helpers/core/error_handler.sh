#!/usr/bin/env bash

setup_error_handler() {
    on_error() {
        local exit_code=$?
        local line_no="${1:-unknown}"
        local command="${2:-unknown}"
        local source_file="${3:-${BASH_SOURCE[1]:-$0}}"
        local function_name="${4:-${FUNCNAME[1]:-main}}"

        local red='' yellow='' dim='' reset=''
        if [[ -t 2 ]]; then
            red=$'\033[1;31m'; yellow=$'\033[1;33m'; dim=$'\033[2m'; reset=$'\033[0m'
        fi
        printf '\n%s╭─ GjallarOS installer could not continue ─╮%s\n' "$red" "$reset" >&2
        printf '%s│%s The last operation failed (exit code %s).%s\n' "$red" "$reset" "$exit_code" "$reset" >&2
        printf '%s│%s No further changes were attempted.%s\n' "$red" "$reset" "$reset" >&2
        printf '%s╰──────────────────────────────────────────╯%s\n' "$red" "$reset" >&2
        printf '\n%sWhat to do next:%s\n' "$yellow" "$reset" >&2
        printf '  1. Read the first error above this summary.\n' >&2
        printf '  2. Correct the reported configuration or hardware issue.\n' >&2
        printf '  3. Run the installer again; existing setup will be preserved.\n' >&2
        printf '\n%s[DEBUG] line %s, function %s, file %s%s\n' \
            "$dim" "$line_no" "$function_name" "$source_file" "$reset" >&2

        exit "$exit_code"
    }

    trap 'on_error "$LINENO" "$BASH_COMMAND" "${BASH_SOURCE[0]}" "${FUNCNAME[0]:-main}"' ERR
}
