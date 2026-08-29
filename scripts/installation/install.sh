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
trap 'say "Installer interrupted by user; no further actions were performed."; exit 130' INT TERM

# ---------------------------------------------------------------------------
# Arguments
# ---------------------------------------------------------------------------

skip_hardware=0
skip_rebuild=0
refresh_hardware=0
existing_install=0
deployment_complete=false

for argument in "$@"; do
    case "$argument" in
        --skip-hardware)
            skip_hardware=1
            ;;

        --no-rebuild)
            skip_rebuild=1
            ;;

        --refresh-hardware)
            refresh_hardware=1
            ;;

        -h|--help)
            printf 'Usage: %s [--skip-hardware] [--refresh-hardware] [--no-rebuild]\n' \
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
load_user_preset "$REPO_ROOT"

if detect_existing_install "$REPO_ROOT"; then
    existing_install=1
    say "Existing GjallarOS installation detected; preserving machine-local setup."
    if [[ -f /etc/nixos/configuration.nix ]] &&
       grep -qE 'services\.fwupd\.enable[[:space:]]*=[[:space:]]*true' /etc/nixos/configuration.nix &&
       grep -q 'catppuccin-gtk' /etc/nixos/configuration.nix &&
       grep -q 'GTK_THEME = "Adwaita:dark"' /etc/nixos/configuration.nix &&
       ( [[ "${cfg_preset_loaded:-false}" == true ]] || command -v zenity >/dev/null 2>&1 ); then
        say "Skipping prerequisite bootstrap."
        ensure_fwupd
    else
        say "Required installer support is missing; bootstrapping fwupd/GTK support now."
        bootstrap_prerequisites
    fi
else
    say "Checking bootstrap prerequisites..."
bootstrap_prerequisites
fi

configure_auto_reboot "$REPO_ROOT"
detect_ui
firmware_update

# ---------------------------------------------------------------------------
# Installer introduction
# ---------------------------------------------------------------------------

say "GjallarOS installer"
say "This wizard writes settings.nix and can generate the selected profile's hardware configuration."

# ---------------------------------------------------------------------------
# Discover available options
# ---------------------------------------------------------------------------

say "Discovering available profiles and components..."

detect_form_factor
say "Detected ${CFG_FORM_FACTOR} hardware; filtering compatible profiles."

mapfile -t profiles < <(
    if [[ "${cfg_preset_loaded:-false}" == true ]]; then
        discover_options "$REPO_ROOT/profiles" directory
    elif [[ "$CFG_FORM_FACTOR" == desktop ]]; then
        [[ -d "$REPO_ROOT/profiles/desktop" ]] && printf '%s\n' desktop
    else
        discover_options "$REPO_ROOT/profiles" directory |
            awk '$0 != "desktop" && $0 != "work" && $0 != "work-user"'
    fi
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

wms=(hyprland)

(( ${#profiles[@]} )) ||
    die "No profiles found."

(( ${#shells[@]} )) ||
    die "No shells found."

(( ${#editors[@]} )) ||
    die "No editors found."

(( ${#browsers[@]} )) ||
    die "No browsers found."

[[ -f "$REPO_ROOT/system/wm/hyprland/default.nix" ]] ||
    die "Hyprland support is missing from this checkout."

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

say "Applying the selected laptop profile..."
configure_framework

say "Configuring Docker..."
configure_docker

say "Configuring local AI..."
configure_ai

say "Detecting graphics..."
detect_graphics

say "Detecting network..."
detect_network

if [[ "$cfg_ai_enable" == true ]]; then
    say "Selecting AI profile..."
    select_ai_profile
else
    say "Local AI disabled; skipping AI hardware profiling."
fi

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

if [[ "$cfg_ai_enable" == true ]]; then
    printf '  local AI: enabled (%s)\n' "$cfg_ai_model"
else
    printf '  local AI: off\n'
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

if [[ "${cfg_preset_loaded:-false}" == true ]]; then
    cfg_write_config="$(preset_bool writeConfig && echo true || echo false)"
else
    confirm "Write this configuration to $REPO_ROOT/settings.nix?" && cfg_write_config=true || cfg_write_config=false
fi
if [[ "$cfg_write_config" != true ]]; then
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

hardware_file="$REPO_ROOT/profiles/$cfg_profile/hardware-configuration.nix"

if (( existing_install && ! refresh_hardware )) && [[ -f "$hardware_file" ]]; then
    skip_hardware=1
    say "Existing hardware configuration found; skipping regeneration."
    say "Use --refresh-hardware to regenerate it deliberately."
elif (( existing_install && ! refresh_hardware )); then
    say "No hardware configuration exists for profile '$cfg_profile'; generating it now."
fi

if (( ! skip_hardware )); then
    say "Generating hardware configuration..."

    generate_hardware_config \
        "$hardware_file"

    say "Configuring TPM2 automatic unlock..."

    configure_tpm2_luks "$hardware_file"

    say "Refreshing settings.nix..."

    render_settings "$REPO_ROOT/settings.nix"

    say "Checking LUKS configuration..."

    # LUKS is intentionally never automated by user.config.json. Even in
    # preset mode, require the operator to make an explicit choice and enter
    # the current key before any key-management action is attempted.
    configure_luks "$hardware_file"
else
    say "Skipping hardware configuration (--skip-hardware)."
    say "Skipping TPM2 and LUKS prompts; existing unlock configuration is unchanged."
fi

# ---------------------------------------------------------------------------
# Protect machine-local configuration from Git
# ---------------------------------------------------------------------------

say "Protecting local machine configuration..."

if (( ! skip_hardware )); then
    protect_local_configs \
        "$REPO_ROOT" \
        "$hardware_file"
else
    protect_local_configs \
        "$REPO_ROOT" \
        ""
fi

# ---------------------------------------------------------------------------
# Rebuild
# ---------------------------------------------------------------------------

if (( ! skip_rebuild )); then
    while true; do
        if [[ "${cfg_preset_loaded:-false}" == true ]]; then
            cfg_run_rebuild="$(preset_bool runRebuild && echo true || echo false)"
            break
        elif confirm "Install the next NixOS generation now (activate after reboot)?"; then
            cfg_run_rebuild=true
            break
        elif [[ "${INSTALLER_UI:-terminal}" == gtk ]]; then
            gtk_cancel_prompt || continue
        else
            cfg_run_rebuild=false
            break
        fi
    done
    if [[ "$cfg_run_rebuild" == true ]]; then
        say "Running NixOS rebuild..."

        run_rebuild "$REPO_ROOT" "$cfg_hostname"

        deployment_complete=true
        say "NixOS generation installed successfully; the current session was left untouched."
        if [[ "$cfg_auto_reboot" == true ]]; then
            say "Automatic reboot was selected; rebooting now."
            sudo systemctl reboot
        elif confirm "Deployment complete. Reboot now?"; then
            say "Rebooting now."
            sudo systemctl reboot
        else
            say "Reboot when convenient to start the new graphical session cleanly."
        fi
    else
        say "Configuration written, but the system was not deployed."
        printf 'Run:\n'
        printf "  sudo nixos-rebuild boot --flake '%s#%s'\n" \
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
if [[ "$deployment_complete" == true ]]; then
    printf '%s\n' 'GjallarOS deployment complete!'
else
    printf '%s\n' 'GjallarOS configuration saved; deployment is still pending.'
fi
printf '%s\n' '============================================================'
printf '\n'
