#!/usr/bin/env bash
set -euo pipefail

rows=(
  "rebuild|Apply the current NixOS configuration.|rebuild"
  "update|Update flake inputs.|update"
  "update --check|Evaluate the flake without changing the system.|update --check"
  "update --rebuild|Update inputs, then rebuild.|update --rebuild"
  "cleanup|Remove old generations and collect garbage.|cleanup"
  "thermal-status|Show CPU temperatures and power state.|thermal-status"
  "thermal-test|Pause/resume a process pattern for troubleshooting.|thermal-test"
  "gjallar-ai|Start the local Ollama assistant.|gjallar-ai"
  "opencode-local|Start OpenCode against local Ollama.|opencode-local"
  "usbguard|Manage USB device policy when USBGuard is enabled.|usbguard list-devices --blocked"
)

show_text() {
  printf 'GjallarOS tools\n\n'
  for row in "${rows[@]}"; do
    IFS='|' read -r command description example <<<"$row"
    printf '  %-18s %s\n    %s\n' "$command" "$description" "$example"
  done
}

if [[ "${1:-}" == "--text" || -z "${DISPLAY:-}${WAYLAND_DISPLAY:-}" ]] || ! command -v yad >/dev/null 2>&1; then
  show_text
  exit 0
fi

args=(
  --list --title='GjallarOS tools' --text='Available commands' --width=900 --height=520
  --center --search-column=1 --column='Command' --column='What it does' --column='Example'
  --button=Close:0 --
)
for row in "${rows[@]}"; do
  IFS='|' read -r command description example <<<"$row"
  args+=("$command" "$description" "$example")
done
exec yad "${args[@]}"
