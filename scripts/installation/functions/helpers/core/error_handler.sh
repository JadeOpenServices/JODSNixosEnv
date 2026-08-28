#!/usr/bin/env bash

setup_error_handler() {
    on_error() {
        local exit_code=$?
        local line_no="${1:-unknown}"
        local command="${2:-unknown}"
        local source_file="${3:-${BASH_SOURCE[1]:-$0}}"
        local function_name="${4:-${FUNCNAME[1]:-main}}"

        printf '\n' >&2
        printf '%s\n' '============================================================' >&2
        printf '%s\n' 'GjallarOS installer failed.' >&2
        printf '%s\n' '------------------------------------------------------------' >&2
        printf 'Exit code : %s\n' "$exit_code" >&2
        printf 'Line      : %s\n' "$line_no" >&2
        printf 'Function  : %s\n' "$function_name" >&2
        printf 'File      : %s\n' "$source_file" >&2
        printf 'Command   : %s\n' "$command" >&2
        printf '%s\n' '============================================================' >&2
        printf '\n' >&2
        printf 'The installer stopped unexpectedly.\n' >&2
        printf 'No further installer actions will be performed.\n' >&2
        printf '\n' >&2

        exit "$exit_code"
    }

    trap 'on_error "$LINENO" "$BASH_COMMAND" "${BASH_SOURCE[0]}" "${FUNCNAME[0]:-main}"' ERR
}