check_nixos_release() {
    local expected actual
    expected="$(sed -n 's/.*"release": "\([0-9.]*\)".*/\1/p' "$REPO_ROOT/deployment/release-policy.json")"
    actual="$(sed -n 's/^VERSION_ID="\([0-9.]*\)"/\1/p' /etc/os-release)"
    [[ -n "$expected" && -n "$actual" ]] || die 'Could not determine the NixOS release.'
    if [[ "$actual" != "$expected" ]]; then
        say "NixOS $actual detected; this checkout targets NixOS $expected."
        confirm "Switch the NixOS channel to $expected and rebuild before continuing?" || die 'Release mismatch not approved.'
        require_command sudo
        require_command nix-channel
        require_command nixos-rebuild
        sudo nix-channel --add "https://channels.nixos.org/nixos-$expected" nixos
        sudo nix-channel --update nixos
        sudo nixos-rebuild switch --upgrade || die "Could not align the system to NixOS $expected."
        actual="$(sed -n 's/^VERSION_ID="\([0-9.]*\)"/\1/p' /run/current-system/etc/os-release 2>/dev/null || true)"
        [[ "$actual" == "$expected" ]] || die "Rebuild completed, but NixOS $expected is not active."
    fi
    say "NixOS release verified: $actual"
}
