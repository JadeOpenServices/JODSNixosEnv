detect_graphics() {
    local pci gpu count bus passthrough_ids

    cfg_graphics_vendor="unknown"
    cfg_graphics_type="integrated"
    cfg_graphics_compute=false
    cfg_graphics_bus_id=""
    cfg_graphics_integrated_bus_id=""
    cfg_graphics_passthrough_ids=""

    command -v lspci >/dev/null 2>&1 || {
        say '[NOTE] lspci unavailable; using generic graphics support.'
        return 0
    }

    # Use -nn rather than -n so we retain both the human-readable
    # controller/vendor names and the numeric PCI IDs.
    pci="$(lspci -Dnn 2>/dev/null || true)"

    gpu="$(
        printf '%s\n' "$pci" |
            grep -Ei 'VGA compatible controller|3D controller|Display controller' ||
            true
    )"

    count="$(printf '%s\n' "$gpu" | sed '/^$/d' | wc -l)"

    if (( count == 0 )); then
        say 'Detected graphics: unknown (integrated)'
        return 0
    fi

    # Determine the GPU vendor.
    if printf '%s\n' "$gpu" | grep -qi nvidia; then
        cfg_graphics_vendor="nvidia"
        cfg_graphics_compute=true
    elif printf '%s\n' "$gpu" | grep -qi amd; then
        cfg_graphics_vendor="amd"
        cfg_graphics_compute=true
    elif printf '%s\n' "$gpu" | grep -qi intel; then
        cfg_graphics_vendor="intel"
    fi

    # Multiple graphics controllers generally means hybrid graphics.
    if (( count > 1 )); then
        cfg_graphics_type="hybrid"
    elif [[ "$cfg_graphics_vendor" == "nvidia" ||
            "$cfg_graphics_vendor" == "amd" ]]; then
        cfg_graphics_type="dedicated"
    else
        cfg_graphics_type="integrated"
    fi

    # Convert:
    #
    #   0000:01:00.0
    #
    # to:
    #
    #   PCI:1:0:0
    #
    bus="$(
        printf '%s\n' "$gpu" |
            head -n1 |
            awk '{print $1}' |
            awk -F'[:.]' '{
                printf "PCI:%d:%d:%d", $2, $3, $4
            }'
    )"

    if [[ "$cfg_graphics_vendor" == "nvidia" ]]; then
        cfg_graphics_bus_id="$bus"
    fi

    # Collect PCI vendor:device IDs for non-Intel GPUs.
    #
    # Example:
    #   10de:2684
    #
    passthrough_ids="$(
        printf '%s\n' "$gpu" |
            grep -Eiv 'Intel' |
            awk '{
                for (i = 1; i <= NF; i++) {
                    if ($i ~ /^[0-9a-fA-F]{4}:[0-9a-fA-F]{4}$/)
                        print $i
                }
            }' |
            sort -u |
            paste -sd' ' -
    )"

    cfg_graphics_passthrough_ids="$passthrough_ids"

    # For hybrid systems, identify the non-NVIDIA controller as the
    # integrated GPU.
    if [[ "$cfg_graphics_type" == "hybrid" ]]; then
        cfg_graphics_integrated_bus_id="$(
            printf '%s\n' "$gpu" |
                grep -Eiv 'nvidia' |
                head -n1 |
                awk '{print $1}' |
                awk -F'[:.]' '{
                    printf "PCI:%d:%d:%d", $2, $3, $4
                }'
        )"
    fi

    say "Detected graphics: ${cfg_graphics_vendor} (${cfg_graphics_type})"
}
