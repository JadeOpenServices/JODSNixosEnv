#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' '== uptime ==' ; uptime
printf '\n%s\n' '== sensors ==' ; sensors 2>/dev/null || true
printf '\n%s\n' '== power profile ==' ; powerprofilesctl get 2>/dev/null || true
printf '\n%s\n' '== top CPU ==' ; ps -eo pid,user,comm,%cpu,%mem --sort=-%cpu | head -n 20
