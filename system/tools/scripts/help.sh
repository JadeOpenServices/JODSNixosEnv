#!/usr/bin/env bash
cat <<'EOF'
GjallarOS tools

  rebuild          Apply the current NixOS configuration.
  update           Update flake inputs.
  update --check   Check the flake without building.
  update --rebuild Update inputs and apply the configuration.
  cleanup          Remove old system generations and collect garbage.
  thermal-status   Show thermal and CPU information.
  thermal-test     Pause/resume a process pattern for troubleshooting.
EOF
