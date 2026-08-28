bootstrap_prerequisites() {
    local config_file=/etc/nixos/configuration.nix
    local backup
    local tmp_file
    local package command
    local packages=()
    local commands=()
    local package_list

    command -v sudo >/dev/null 2>&1 ||
        die 'sudo is required to install missing NixOS prerequisites.'

    declare -A requirements=(
        [lspci]=pciutils
        [git]=git
        [fwupdmgr]=fwupd
        [curl]=curl
    )
    # Preset mode is deliberately terminal-only, so it does not need GTK.
    [[ "${cfg_preset_loaded:-false}" == true ]] || requirements[zenity]=zenity
    if [[ -f "$config_file" ]] && ! grep -q 'catppuccin-gtk' "$config_file"; then
        packages+=(catppuccin-gtk)
        commands+=("catppuccin-gtk theme")
    fi
    if [[ -f "$config_file" ]] && ! grep -q 'GTK_THEME' "$config_file"; then
        case " ${packages[*]} " in
            *" catppuccin-gtk "*) ;;
            *) packages+=(catppuccin-gtk) ;;
        esac
        commands+=("GTK_THEME")
    fi

    for command in "${!requirements[@]}"; do
        if ! command -v "$command" >/dev/null 2>&1; then
            commands+=("$command")
            package="${requirements[$command]}"

            case " ${packages[*]} " in
                *" $package "*) ;;
                *) packages+=("$package") ;;
            esac
        fi
    done

    # Installing the fwupd package alone does not register its daemon; the
    # NixOS module must also be enabled in configuration.nix.
    if [[ -f "$config_file" ]] &&
       ! grep -qE 'services\.fwupd\.enable[[:space:]]*=[[:space:]]*true' "$config_file"; then
        case " ${packages[*]} " in
            *" fwupd "*) ;;
            *) packages+=(fwupd); commands+=("services.fwupd.enable") ;;
        esac
    fi

    if (( ${#packages[@]} == 0 )); then
        ensure_fwupd
        return 0
    fi

    printf '\nMissing helper commands: %s\n' "${commands[*]}"
    printf 'The installer can add these packages to %s:\n  %s\n' \
        "$config_file" "${packages[*]}"

    confirm 'Show the proposed change and install it with nixos-rebuild?' || {
        say 'Continuing without optional helper packages.'
        return 0
    }

    [[ -f "$config_file" ]] ||
        die "$config_file does not exist; install the standard NixOS configuration first."

    backup="${config_file}.gjallar-backup.$(date +%Y%m%d%H%M%S)"
    tmp_file="${config_file}.gjallar-tmp"
    package_list="${packages[*]}"

    sudo cp -a -- "$config_file" "$backup"

    sudo awk -v packages="$package_list" '
        #
        # Read the ENTIRE file first.
        # This is important: we must preserve every existing line.
        #
        {
            lines[NR] = $0
        }

        END {
            system_packages_line = 0
            system_packages_end = 0

            #
            # Find environment.systemPackages.
            #
            for (i = 1; i <= NR; i++) {
                if (lines[i] ~ /^[[:space:]]*environment\.systemPackages[[:space:]]*=/) {
                    system_packages_line = i

                    #
                    # Single-line form:
                    #
                    # environment.systemPackages = with pkgs; [ foo bar ];
                    #
                    if (lines[i] ~ /\[[^]]*\][[:space:]]*;/) {
                        system_packages_end = i
                        break
                    }

                    #
                    # Multiline form. Find the closing ];.
                    #
                    for (j = i + 1; j <= NR; j++) {
                        if (lines[j] ~ /^[[:space:]]*\][[:space:]]*;/) {
                            system_packages_end = j
                            break
                        }
                    }

                    break
                }
            }

            #
            # No existing environment.systemPackages.
            #
            if (system_packages_line == 0) {
                inserted = 0

                for (i = 1; i <= NR; i++) {
                    if (!inserted && lines[i] ~ /^}[[:space:]]*$/) {
                        print "    environment.systemPackages = with pkgs; ["
                        
                        n = split(packages, wanted, /[[:space:]]+/)

                        for (j = 1; j <= n; j++) {
                            if (wanted[j] != "")
                                print "      " wanted[j]
                        }

                        print "    ];"
                        inserted = 1
                    }

                    print lines[i]
                }

                if (!inserted)
                    exit 2

                exit 0
            }

            #
            # Existing package list.
            #
            #
            # Build a string containing the existing package-list text.
            #
            existing = ""

            for (i = system_packages_line; i <= system_packages_end; i++)
                existing = existing " " lines[i]

            #
            # Determine which requested packages are not already present.
            #
            additions = ""

            n = split(packages, wanted, /[[:space:]]+/)

            for (i = 1; i <= n; i++) {
                package = wanted[i]

                if (package == "")
                    continue

                #
                # Check for the package as a whitespace-delimited token.
                # This avoids treating e.g. "fwupd" as present because
                # some unrelated identifier happens to contain that text.
                #
                padded = " " existing " "

                if (padded !~ "[[:space:]]" package "[[:space:]]") {
                    additions = additions package " "
                }
            }

            #
            # Nothing new to add. Just reproduce the original file.
            #
            if (additions == "") {
                for (i = 1; i <= NR; i++)
                    print lines[i]

                exit 0
            }

            #
            # Single-line package list.
            #
            if (system_packages_line == system_packages_end) {
                line = lines[system_packages_line]

                sub(/\][[:space:]]*;/,
                    " " additions "];",
                    line)

                for (i = 1; i <= NR; i++) {
                    if (i == system_packages_line)
                        print line
                    else
                        print lines[i]
                }

                exit 0
            }

            #
            # Multiline package list.
            #
            for (i = 1; i <= NR; i++) {
                if (i == system_packages_end) {
                    n = split(additions, add, /[[:space:]]+/)

                    for (j = 1; j <= n; j++) {
                        if (add[j] != "")
                            print "    " add[j]
                    }
                }

                print lines[i]
            }
        }
    ' "$config_file" | sudo tee "$tmp_file" >/dev/null || {
        sudo rm -f -- "$tmp_file"
        die 'Could not update configuration.nix.'
    }

    sudo mv -- "$tmp_file" "$config_file"

    if ! sudo grep -qE 'services\.fwupd\.enable[[:space:]]*=[[:space:]]*true' "$config_file"; then
        tmp_file="${config_file}.gjallar-tmp"
        sudo awk '
            ! inserted && (/^[[:space:]]*\{[[:space:]]*$/ || /:[[:space:]]*\{[[:space:]]*$/) {
                print
                print "    services.fwupd.enable = true;"
                inserted = 1
                next
            }
            { print }
            END { if (!inserted) exit 2 }
        ' "$config_file" | sudo tee "$tmp_file" >/dev/null || {
            sudo rm -f -- "$tmp_file"
            die 'Could not enable services.fwupd in configuration.nix.'
        }
        sudo mv -- "$tmp_file" "$config_file"
        say 'Enabled services.fwupd in configuration.nix.'
    fi

    if ! sudo grep -q 'GTK_THEME' "$config_file"; then
        tmp_file="${config_file}.gjallar-tmp"
        sudo awk '
            ! inserted && (/^[[:space:]]*\{[[:space:]]*$/ || /:[[:space:]]*\{[[:space:]]*$/) {
                print
                print "    environment.variables.GTK_THEME = \"Adwaita:dark\";"
                inserted = 1
                next
            }
            { print }
            END { if (!inserted) exit 2 }
        ' "$config_file" | sudo tee "$tmp_file" >/dev/null || {
            sudo rm -f -- "$tmp_file"
            die 'Could not enable the Catppuccin GTK theme in configuration.nix.'
        }
        sudo mv -- "$tmp_file" "$config_file"
        say 'Enabled the Catppuccin GTK theme system-wide.'
    fi
    sudo sed -i 's/GTK_THEME[[:space:]]*=[[:space:]]*"[^"]*"/GTK_THEME = "Adwaita:dark"/' "$config_file"

    say "Backed up configuration to $backup"

    if sudo nixos-rebuild switch; then
        sudo rm -f -- "$backup"

        ensure_fwupd

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
