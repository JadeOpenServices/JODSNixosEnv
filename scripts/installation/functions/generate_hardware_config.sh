generate_hardware_config() {
    local target="$1" backup
    require_command nixos-generate-config
    if [[ -e "$target" ]]; then
        backup="${target}.bak.$(date +%Y%m%d%H%M%S)"
        cp -- "$target" "$backup"
        say "Backed up existing hardware configuration to $backup"
    fi
    nixos-generate-config --show-hardware-config > "$target"
    say "Generated hardware configuration: $target"
}
