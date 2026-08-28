detect_form_factor() {
    local chassis_type product
    CFG_FORM_FACTOR="laptop"

    # A battery is the most reliable portable-machine signal. DMI chassis
    # types provide a fallback for systems whose battery driver is unavailable.
    if compgen -G '/sys/class/power_supply/BAT*' >/dev/null 2>&1; then
        CFG_FORM_FACTOR="laptop"
    else
        chassis_type="$(< /sys/class/dmi/id/chassis_type 2>/dev/null || true)"
        case "$chassis_type" in
            3|4|5|6|7|15|16|17|18|19|20|21|22|23|24|25|26|27|28|29|30|31|32)
                CFG_FORM_FACTOR="desktop"
                ;;
        esac
    fi

    product="$(< /sys/class/dmi/id/product_name 2>/dev/null || true)"
    if [[ "$product" =~ [Tt]hink[Pp]ad ]]; then
        CFG_FORM_FACTOR="laptop"
        CFG_LAPTOP_VENDOR="thinkpad"
    elif [[ "$product" =~ [Ff]ramework ]]; then
        CFG_FORM_FACTOR="laptop"
        CFG_LAPTOP_VENDOR="framework"
    else
        CFG_LAPTOP_VENDOR="generic"
    fi
}
