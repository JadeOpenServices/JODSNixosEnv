resolve_backgrounds() {
    local repo_root="$1" dotfiles_dir="$2" role value url_hash extension target
    local -a roles=(normal work gaming)

    for role in "${roles[@]}"; do
        local variable="cfg_background_${role}"
        value="${!variable:-}"
        [[ -n "$value" ]] || continue

        if [[ "$value" =~ ^https:// ]]; then
            require_command curl
            require_command sha256sum
            require_command cut

            url_hash="$(printf '%s' "$value" | sha256sum | cut -c1-16)"
            extension="${value%%\?*}"
            extension="${extension##*/}"
            extension="${extension##*.}"
            [[ "$extension" =~ ^(png|jpg|jpeg|webp|avif|svg)$ ]] || extension="png"

            target="$repo_root/non-nix/wallpapers/user-${role}-${url_hash}.${extension}"
            if [[ ! -s "$target" ]]; then
                say "Downloading ${role} background..."
                mkdir -p "$(dirname "$target")"
                curl --fail --location --retry 3 --proto '=https' --tlsv1.2 \
                    --output "$target" "$value" || {
                    rm -f -- "$target"
                    die "Could not download ${role} background from HTTPS URL."
                }
            fi
            printf -v "$variable" '%s' "$target"
        elif [[ "$value" != /* ]]; then
            # Relative paths are interpreted from the configured repository.
            printf -v "$variable" '%s/%s' "$dotfiles_dir" "$value"
        fi
    done
}
