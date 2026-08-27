#!/usr/bin/env bash
# Interactive AlfheimOS configuration and installation wizard.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
FUNCTIONS_DIR="$SCRIPT_DIR/functions"

for function_file in "$FUNCTIONS_DIR"/*.sh; do
    # shellcheck source=/dev/null
    source "$function_file"
done

skip_hardware=0
skip_rebuild=0
for argument in "$@"; do
    case "$argument" in
        --skip-hardware) skip_hardware=1 ;;
        --no-rebuild) skip_rebuild=1 ;;
        -h|--help)
            printf 'Usage: %s [--skip-hardware] [--no-rebuild]\n' "${0##*/}"
            exit 0
            ;;
        *) die "Unknown option: $argument" ;;
    esac
done

require_command awk
require_command sed
require_command find
require_command grep
is_nixos || die "This installer must be run from a NixOS system."
check_nixos_release
bootstrap_prerequisites

say "AlfheimOS installer"
say "This wizard writes settings.nix and can generate the selected profile's hardware configuration."

mapfile -t profiles < <(discover_options "$REPO_ROOT/profiles" directory)
mapfile -t shells < <(discover_options "$REPO_ROOT/user/shells" nix-file)
mapfile -t editors < <(discover_options "$REPO_ROOT/user/editors" directory)
mapfile -t browsers < <(discover_options "$REPO_ROOT/user/browsers" nix-file)
mapfile -t themes < <(discover_options "$REPO_ROOT/themes" nix-file)
mapfile -t user_wms < <(discover_options "$REPO_ROOT/user/wm" directory)

wms=()
for wm in "${user_wms[@]}"; do
    [[ -f "$REPO_ROOT/system/wm/$wm/default.nix" ]] && wms+=("$wm")
done

(( ${#profiles[@]} )) || die "No profiles found."
(( ${#shells[@]} )) || die "No shells found."
(( ${#editors[@]} )) || die "No editors found."
(( ${#browsers[@]} )) || die "No browsers found."
(( ${#wms[@]} )) || die "No shared window managers found."
(( ${#themes[@]} )) || die "No themes found."

collect_settings "$REPO_ROOT" profiles shells editors browsers wms themes
detect_hardware
configure_framework
configure_scrobbling "$REPO_ROOT" "$cfg_username"
configure_nemu

printf '\nSelected configuration:\n'
printf '  profile: %s\n  hostname: %s\n  user: %s\n  shell: %s\n  theme: %s\n' \
    "$cfg_profile" "$cfg_hostname" "$cfg_username" "$cfg_shell" "$cfg_theme"
printf '  editors: %s\n  browsers: %s\n  window managers: %s\n' \
    "${cfg_editors[*]}" "${cfg_browsers[*]}" "${cfg_wms[*]}"
printf '  framework: %s  nemu: %s  wine: system-wide WoW64 staging\n' \
    "$([[ "$cfg_framework_enable" == true ]] && printf '%s' "Framework $cfg_framework_model" || printf 'off')" \
    "$([[ "$cfg_nemu_enable" == true ]] && printf 'enabled' || printf 'off')"
[[ "$cfg_work_user_enable" == true ]] && printf '  work account: %s (no gaming modules)\n' "$cfg_work_username"

confirm "Write this configuration to $REPO_ROOT/settings.nix?" || {
    say "Nothing was changed."
    exit 0
}

render_settings "$REPO_ROOT/settings.nix"
if [[ "$cfg_enable_scrobbling" == true ]]; then
    check_sops "$REPO_ROOT" "$cfg_username"
fi

if (( ! skip_hardware )); then
    generate_hardware_config "$REPO_ROOT/profiles/$cfg_profile/hardware-configuration.nix"
    configure_tpm2_luks "$REPO_ROOT/profiles/$cfg_profile/hardware-configuration.nix"
    render_settings "$REPO_ROOT/settings.nix"
    configure_luks "$REPO_ROOT/profiles/$cfg_profile/hardware-configuration.nix"
fi

if (( ! skip_rebuild )); then
    if confirm "Run nixos-rebuild switch now?"; then
        run_rebuild "$REPO_ROOT" "$cfg_hostname"
    else
        say "Configuration written. Run: sudo nixos-rebuild switch --flake '$REPO_ROOT#$cfg_hostname'"
    fi
fi
