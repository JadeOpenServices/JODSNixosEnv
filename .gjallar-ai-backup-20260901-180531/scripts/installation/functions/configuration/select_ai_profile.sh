#!/usr/bin/env bash

cfg_ai_enable=true

system_ram_gb=0
gpu_vendor="unknown"
gpu_type="unknown"
gpu_vram_mb=0

if [[ -r /proc/meminfo ]]; then
    mem_kb="$(awk '/^MemTotal:/ {print $2}' /proc/meminfo)"
    if [[ -n "$mem_kb" ]]; then
        system_ram_gb="$((mem_kb / 1024 / 1024))"
    fi
fi

if command -v lspci >/dev/null 2>&1; then
    gpu_lines="$(lspci -nn 2>/dev/null | grep -Ei 'VGA compatible controller|3D controller|Display controller' || true)"

    if grep -qi 'AMD' <<< "$gpu_lines"; then
        gpu_vendor="amd"
    elif grep -qi 'NVIDIA' <<< "$gpu_lines"; then
        gpu_vendor="nvidia"
    elif grep -qi 'Intel' <<< "$gpu_lines"; then
        gpu_vendor="intel"
    fi

    gpu_count="$(printf '%s\n' "$gpu_lines" | grep -c . || true)"

    if (( gpu_count > 1 )); then
        gpu_type="dedicated"
    elif grep -qiE 'AMD.*Phoenix|AMD.*Radeon.*Graphics|Intel.*Iris|Intel.*UHD' <<< "$gpu_lines"; then
        gpu_type="integrated"
    elif [[ "$gpu_vendor" != "unknown" ]]; then
        gpu_type="integrated"
    fi
fi

for vram_file in /sys/class/drm/card*/device/mem_info_vram_total(N); do
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

if (( system_ram_gb >= 32 && gpu_vram_mb >= 12288 && gpu_type == dedicated )); then
    cfg_ai_model="qwen3-coder:30b"
    cfg_ai_context_tokens=32768
    cfg_ai_vram_mb="$gpu_vram_mb"
    cfg_ai_profile="dedicated"
elif (( system_ram_gb >= 16 )); then
    cfg_ai_model="qwen3-coder:14b"
    cfg_ai_context_tokens=16384
    cfg_ai_vram_mb="$gpu_vram_mb"
    cfg_ai_profile="integrated"
else
    cfg_ai_model="qwen3-coder:7b"
    cfg_ai_context_tokens=8192
    cfg_ai_vram_mb="$gpu_vram_mb"
    cfg_ai_profile="low-memory"
fi

if (( gpu_vram_mb == 0 )); then
    cfg_ai_vram_mb=0
fi

printf 'AI hardware evaluation:\n'
printf '  RAM:     %s GB\n' "$system_ram_gb"
printf '  GPU:     %s\n' "$gpu_vendor"
printf '  Type:    %s\n' "$gpu_type"
printf '  VRAM:    %s MB\n' "$cfg_ai_vram_mb"
printf '  Profile: %s\n' "$cfg_ai_profile"
printf '  Model:   %s\n' "$cfg_ai_model"
printf '  Context: %s\n' "$cfg_ai_context_tokens"
