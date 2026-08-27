#!/usr/bin/env bash
# Interactive GjallarOS configuration and installation wizard.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
FUNCTIONS_DIR="$SCRIPT_DIR/functions"

# ---------------------------------------------------------------------------
# Bootstrap commands
# ---------------------------------------------------------------------------

for bootstrap_command in find sort; do
    command -v "$bootstrap_command" >/dev/null 2>&1 || {
        printf 'Error: required command not found: %s\n' \
            "$bootstrap_command" >&2
        exit 1
    }
done

# ---------------------------------------------------------------------------
# Load installer functions
# ---------------------------------------------------------------------------

while IFS= read -r -d '' function_file; do
    # shellcheck source=/dev/null
    source "$function_file"
done < <(
    find "$FUNCTIONS_DIR" \
        -type f \
        -name '*.sh' \
        -print0 |
        sort -z
)

# Enable the global error handler after all functions are loaded.
setup_error_handler

# ---------------------------------------------------------------------------
# Arguments
# ---------------------------------------------------------------------------

skip_hardware=0
skip_rebuild=0

for argument in "$@"; do
    case "$argument" in
        --skip-hardware)
            skip_hardware=1
            ;;

        --no-rebuild)
            skip_rebuild=1
            ;;

        -h|--help)
            printf 'Usage: %s [--skip-hardware] [--no-rebuild]\n' \
                "${0##*/}"
            exit 0
            ;;

        *)
            die "Unknown option: $argument"
            ;;
    esac
done

# ---------------------------------------------------------------------------
# Prerequisites
# ---------------------------------------------------------------------------

say "Checking installer prerequisites..."

require_command awk
require_command sed
require_command find
require_command grep

say "Checking NixOS installation..."

if ! is_nixos; then
    die "This installer must be run from a NixOS system."
fi

say "Checking NixOS release..."
check_nixos_release

say "Checking bootstrap prerequisites..."
bootstrap_prerequisites

# ---------------------------------------------------------------------------
# Installer introduction
# ---------------------------------------------------------------------------

say "GjallarOS installer"
say "This wizard writes settings.nix and can generate the selected profile's hardware configuration."

# ---------------------------------------------------------------------------
# Discover available options
# ---------------------------------------------------------------------------

say "Discovering available profiles and components..."

mapfile -t profiles < <(
    discover_options "$REPO_ROOT/profiles" directory
)

mapfile -t shells < <(
    discover_options "$REPO_ROOT/user/shells" nix-file
)

mapfile -t editors < <(
    discover_options "$REPO_ROOT/user/editors" directory
)

mapfile -t browsers < <(
    discover_options "$REPO_ROOT/user/browsers" nix-file
)

mapfile -t themes < <(
    discover_options "$REPO_ROOT/themes" nix-file
)

mapfile -t user_wms < <(
    discover_options "$REPO_ROOT/user/wm" directory
)

wms=()

for wm in "${user_wms[@]}"; do
    if [[ -f "$REPO_ROOT/system/wm/$wm/default.nix" ]]; then
        wms+=("$wm")
    fi
done

(( ${#profiles[@]} )) ||
    die "No profiles found."

(( ${#shells[@]} )) ||
    die "No shells found."

(( ${#editors[@]} )) ||
    die "No editors found."

(( ${#browsers[@]} )) ||
    die "No browsers found."

(( ${#wms[@]} )) ||
    die "No shared window managers found."

(( ${#themes[@]} )) ||
    die "No themes found."

# ---------------------------------------------------------------------------
# Collect configuration
# ---------------------------------------------------------------------------

say "Collecting system configuration..."

collect_settings \
    "$REPO_ROOT" \
    profiles \
    shells \
    editors \
    browsers \
    wms \
    themes

say "Configuring Docker..."
configure_docker

say "Detecting graphics..."
detect_graphics

say "Detecting network..."
detect_network

say "Selecting AI profile..."
select_ai_profile

say "Configuring Framework..."
configure_framework

say "Configuring scrobbling..."
configure_scrobbling \
    "$REPO_ROOT" \
    "$cfg_username"

say "Configuring NEMU..."
configure_nemu

# ---------------------------------------------------------------------------
# Configuration summary
# ---------------------------------------------------------------------------

printf '\n'
printf '%s\n' '============================================================'
printf '%s\n' 'Selected configuration'
printf '%s\n' '============================================================'

printf '  profile: %s\n' "$cfg_profile"
printf '  hostname: %s\n' "$cfg_hostname"
printf '  user: %s\n' "$cfg_username"
printf '  shell: %s\n' "$cfg_shell"
printf '  theme: %s\n' "$cfg_theme"

printf '  editors: %s\n' "${cfg_editors[*]}"
printf '  browsers: %s\n' "${cfg_browsers[*]}"
printf '  window managers: %s\n' "${cfg_wms[*]}"

if [[ "$cfg_framework_enable" == true ]]; then
    printf '  framework: Framework %s\n' "$cfg_framework_model"
else
    printf '  framework: off\n'
fi

if [[ "$cfg_nemu_enable" == true ]]; then
    printf '  nemu: enabled\n'
else
    printf '  nemu: off\n'
fi

if [[ "$cfg_work_user_enable" == true ]]; then
    printf '  work account: %s (no gaming modules)\n' \
        "$cfg_work_username"
fi

printf '%s\n' '============================================================'
printf '\n'

# ---------------------------------------------------------------------------
# Write settings
# ---------------------------------------------------------------------------

if ! confirm "Write this configuration to $REPO_ROOT/settings.nix?"; then
    say "Nothing was changed."
    exit 0
fi

say "Writing settings.nix..."

render_settings "$REPO_ROOT/settings.nix"

if [[ "$cfg_enable_scrobbling" == true ]]; then
    say "Checking SOPS configuration..."
    check_sops "$REPO_ROOT" "$cfg_username"
fi

# ---------------------------------------------------------------------------
# Hardware configuration
# ---------------------------------------------------------------------------

if (( ! skip_hardware )); then
    say "Generating hardware configuration..."

    generate_hardware_config \
        "$REPO_ROOT/profiles/$cfg_profile/hardware-configuration.nix"

    say "Configuring TPM2/LUKS..."

    configure_tpm2_luks \
        "$REPO_ROOT/profiles/$cfg_profile/hardware-configuration.nix"

    say "Refreshing settings.nix..."

    render_settings "$REPO_ROOT/settings.nix"

    say "Configuring LUKS..."

    configure_luks \
        "$REPO_ROOT/profiles/$cfg_profile/hardware-configuration.nix"
else
    say "Skipping hardware configuration (--skip-hardware)."
fi

# ---------------------------------------------------------------------------
# Rebuild
# ---------------------------------------------------------------------------

if (( ! skip_rebuild )); then
    if confirm "Run nixos-rebuild switch now?"; then
        say "Running NixOS rebuild..."

        run_rebuild "$REPO_ROOT" "$cfg_hostname"

        say "NixOS rebuild completed successfully."
    else
        say "Configuration written."
        printf 'Run:\n'
        printf "  sudo nixos-rebuild switch --flake '%s#%s'\n" \
            "$REPO_ROOT" \
            "$cfg_hostname"
    fi
else
    say "Skipping rebuild (--no-rebuild)."
fi

# ---------------------------------------------------------------------------
# Complete
# ---------------------------------------------------------------------------

printf '\n'
printf '%s\n' '============================================================'
printf '%s\n' 'GjallarOS configuration complete!'
printf '%s\n' '============================================================'
printf '\n'