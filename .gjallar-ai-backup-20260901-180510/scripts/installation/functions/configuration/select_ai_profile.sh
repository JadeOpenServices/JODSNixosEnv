select_ai_profile() {
    local ram_gb vram_mb vram_bytes vram_file
    cfg_ai_model="qwen3-coder:30b"
    cfg_ai_context_tokens=8192
    cfg_ai_vram_mb=0
    ram_gb="$(( $(awk '/MemTotal:/ {print $2}' /proc/meminfo 2>/dev/null || printf 8192) / 1024 / 1024 ))"
    vram_mb=0
    if command -v nvidia-smi >/dev/null 2>&1; then
        vram_mb="$(nvidia-smi --query-gpu=memory.total --format=csv,noheader,nounits 2>/dev/null | awk 'BEGIN{m=0} {if ($1 > m) m=$1} END{print m+0}')"
    fi
    for vram_file in /sys/class/drm/card*/device/mem_info_vram_total; do
        [[ -r "$vram_file" ]] || continue
        vram_bytes="$(cat "$vram_file" 2>/dev/null || printf 0)"
        (( vram_bytes / 1024 / 1024 > vram_mb )) && vram_mb=$((vram_bytes / 1024 / 1024))
    done
    cfg_ai_vram_mb="$vram_mb"
    if [[ "$ram_gb" -lt 12 || ( "$cfg_graphics_type" == dedicated && "$vram_mb" -lt 4096 ) ]]; then
        cfg_ai_model="qwen3-coder:30b"; cfg_ai_context_tokens=4096
    elif [[ "$cfg_graphics_type" == dedicated && "$vram_mb" -ge 12288 && "$ram_gb" -ge 24 ]]; then
        cfg_ai_model="qwen3-coder:30b"; cfg_ai_context_tokens=16384
    fi
    say "Selected local AI profile: $cfg_ai_model (${ram_gb} GiB RAM, ${vram_mb} MiB VRAM detected)"
}
