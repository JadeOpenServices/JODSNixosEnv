bootstrap_prerequisites() {
    local config_file=/etc/nixos/configuration.nix backup package command packages=() commands=()
    command -v sudo >/dev/null 2>&1 || die 'sudo is required to install missing NixOS prerequisites.'
    declare -A requirements=(
        [lspci]=pciutils
        [git]=git
        [fwupdmgr]=fwupd
    )
    for command in "${!requirements[@]}"; do
        command -v "$command" >/dev/null 2>&1 || {
            commands+=("$command")
            package="${requirements[$command]}"
            [[ " ${packages[*]} " == *" $package "* ]] || packages+=("$package")
        }
    done
    if (( ${#packages[@]} == 0 )); then
        if command -v fwupdmgr >/dev/null 2>&1 && command -v systemctl >/dev/null 2>&1; then
            sudo systemctl enable --now fwupd.service 2>/dev/null || say 'fwupd is installed but could not be started yet.'
        fi
        return 0
    fi

    printf '\nMissing helper commands: %s\n' "${commands[*]}"
    printf 'The installer can add these packages to %s:\n  %s\n' "$config_file" "${packages[*]}"
    confirm 'Show the proposed change and install it with nixos-rebuild?' || {
        say 'Continuing without optional helper packages.'
        return 0
    }
    [[ -f "$config_file" ]] || die "$config_file does not exist; install the standard NixOS configuration first."
    backup="${config_file}.alfheim-backup.$(date +%Y%m%d%H%M%S)"
    sudo cp -a -- "$config_file" "$backup"
    sudo awk -v packages="${packages[*]}" '
        BEGIN { inserted = 0 }
        { lines[NR] = $0 }
        END {
            for (i = 1; i <= NR; i++) {
                if (i == NR && lines[i] ~ /^}[[:space:]]*$/ && !inserted) {
                    print "    environment.systemPackages = with pkgs; [ " packages " ];"
                    inserted = 1
                }
                print lines[i]
            }
            if (!inserted) exit 2
        }
    ' "$config_file" | sudo tee "$config_file.tmp" >/dev/null || die 'Could not update configuration.nix.'
    sudo mv -- "$config_file.tmp" "$config_file"
    say "Backed up configuration to $backup"
    if sudo nixos-rebuild switch; then
        sudo rm -f -- "$backup"
        sudo systemctl enable --now fwupd.service 2>/dev/null || say 'fwupd was installed, but its service could not be started.'
        say 'Bootstrap rebuild succeeded; removed the temporary configuration backup.'
    else
        say 'Bootstrap rebuild failed; restoring the previous configuration.' >&2
        if sudo cp -a -- "$backup" "$config_file"; then
            sudo rm -f -- "$backup"
            say 'Previous configuration restored.' >&2
        else
            say "Could not restore the previous configuration; backup remains at $backup" >&2
        fi
        return 1
    fi
}
