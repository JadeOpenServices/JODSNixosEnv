generate_hardware_config() {
    local target="$1" backup temporary
    require_command nixos-generate-config
    require_command mktemp
    mkdir -p -- "$(dirname "$target")"
    if [[ -e "$target" ]]; then
        backup="${target}.bak.$(date +%Y%m%d%H%M%S)"
        cp -- "$target" "$backup"
        say "Backed up existing hardware configuration to $backup"
    fi
    temporary="$(mktemp "${target}.tmp.XXXXXX")"
    if ! nixos-generate-config --show-hardware-config > "$temporary"; then
        rm -f -- "$temporary"
        die 'Could not generate hardware configuration; the existing file was preserved.'
    fi
    mv -- "$temporary" "$target"
    say "Generated hardware configuration: $target"
}
