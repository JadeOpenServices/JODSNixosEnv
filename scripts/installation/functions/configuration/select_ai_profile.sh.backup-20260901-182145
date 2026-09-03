#!/usr/bin/env bash

cfg_ai_enable=true

system_ram_gb=0
gpu_vendor="unknown"
gpu_type="unknown"
gpu_vram_mb=0

if [[ -r /proc/meminfo ]]; then
    mem_kb="$(awk '/^MemTotal:/ {print $2}' /proc/meminfo)"

    if [[ "$mem_kb" =~ ^[0-9]+$ ]]; then
        system_ram_gb="$((mem_kb / 1024 / 1024))"
    fi
fi

if command -v lspci >/dev/null 2>&1; then
    gpu_lines="$(lspci -nn 2>/dev/null | grep -Ei 'VGA compatible controller|3D controller|Display controller' || true)"

    if printf '%s\n' "$gpu_lines" | grep -qi 'AMD'; then
        gpu_vendor="amd"
    elif printf '%s\n' "$gpu_lines" | grep -qi 'NVIDIA'; then
        gpu_vendor="nvidia"
    elif printf '%s\n' "$gpu_lines" | grep -qi 'Intel'; then
        gpu_vendor="intel"
    fi

    gpu_count="$(printf '%s\n' "$gpu_lines" | grep -c . || true)"

    if (( gpu_count > 1 )); then
        gpu_type="dedicated"
    elif printf '%s\n' "$gpu_lines" | grep -qiE 'AMD.*Phoenix|AMD.*Radeon.*Graphics|Intel.*Iris|Intel.*UHD'; then
        gpu_type="integrated"
    elif [[ "$gpu_vendor" != "unknown" ]]; then
        gpu_type="integrated"
    fi
fi

for vram_file in /sys/class/drm/card*/device/mem_info_vram_total; do
    if [[ -r "$vram_file" ]]; then
        vram_bytes="$(cat "$vram_file" 2>/dev/null)"

        if [[ "$vram_bytes" =~ ^[0-9]+$ ]]; then
            vram_mb="$((vram_bytes / 1024 / 1024))"

            if (( vram_mb > gpu_vram_mb )); then
                gpu_vram_mb="$vram_mb"
            fi
        fi
    fi
done

if (( gpu_vram_mb == 0 )); then
    gpu_vram_mb=0
fi

cfg_ai_profile="low-memory"
cfg_ai_model="qwen3-coder:7b"
cfg_ai_context_tokens=8192
cfg_ai_vram_mb="$gpu_vram_mb"

if (( system_ram_gb >= 32 )) &&
   [[ "$gpu_type" == "dedicated" ]] &&
   (( gpu_vram_mb >= 12288 )); then

    cfg_ai_profile="dedicated"
    cfg_ai_model="qwen3-coder:30b"
    cfg_ai_context_tokens=32768

elif (( system_ram_gb >= 16 )); then

    cfg_ai_profile="integrated"
    cfg_ai_model="qwen3-coder:14b"
    cfg_ai_context_tokens=16384
fi

user_config="$repo_root/user.config.json"
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../../../.." 2>/dev/null && pwd)"

override_ai_selection=false
override_model_with=""

if [[ -f "$repo_root/user.config.json" ]] &&
   command -v python3 >/dev/null 2>&1; then

    override_values="$(
        python3 - "$repo_root/user.config.json" <<'PYCONFIG'
import json
import shlex
import sys

try:
    with open(sys.argv[1], "r", encoding="utf-8") as handle:
        data = json.load(handle)
except Exception:
    print("override_ai_selection=false")
    print("override_model_with=''")
    raise SystemExit(0)

enabled = data.get("overrideAiSelection", False)

if isinstance(enabled, bool):
    enabled = enabled
else:
    enabled = str(enabled).strip().lower() == "true"

print(
    "override_ai_selection=" +
    ("true" if enabled else "false")
)

if enabled:
    model = data.get("overrideModelWith", "")

    if isinstance(model, str):
        model = model.strip()
    else:
        model = ""

    print(
        "override_model_with=" +
        shlex.quote(model)
    )
else:
    print("override_model_with=''")
PYCONFIG
    )"

    eval "$override_values"
fi

if [[ "$override_ai_selection" == "true" ]] &&
   [[ -n "$override_model_with" ]]; then

    cfg_ai_profile="user-override"
    cfg_ai_model="$override_model_with"

    if (( system_ram_gb >= 32 )); then
        cfg_ai_context_tokens=32768
    elif (( system_ram_gb >= 16 )); then
        cfg_ai_context_tokens=16384
    else
        cfg_ai_context_tokens=8192
    fi

    printf '%s\n' 'AI model override enabled.'
    printf '  overrideModelWith: %s\n' "$cfg_ai_model"
else
    if [[ "$override_ai_selection" == "true" ]]; then
        printf '%s\n' 'AI model override requested but overrideModelWith is empty.'
        printf '%s\n' 'Falling back to automatic hardware-aware selection.'
    fi
fi

printf '%s\n' 'AI hardware evaluation:'
printf '  RAM:     %s GB\n' "$system_ram_gb"
printf '  GPU:     %s\n' "$gpu_vendor"
printf '  Type:    %s\n' "$gpu_type"
printf '  VRAM:    %s MB\n' "$cfg_ai_vram_mb"
printf '  Profile: %s\n' "$cfg_ai_profile"
printf '  Model:   %s\n' "$cfg_ai_model"
printf '  Context: %s\n' "$cfg_ai_context_tokens"
