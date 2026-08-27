check_sops() {
    local repo_root="$1" username="$2" home_dir key_file
    home_dir="$(user_home_dir "$username")"
    key_file="$home_dir/.config/sops/age/keys.txt"
    if [[ ! -f "$repo_root/secrets/default.yaml" ]]; then
        say "SOPS: no secrets/default.yaml found; continuing without secrets."
        return
    fi
    if [[ -f "$key_file" ]]; then
        say "SOPS: age key found at $key_file"
    else
        say "SOPS: age key missing at $key_file"
        say "Create it before rebuilding: mkdir -p \"$(dirname "$key_file")\" && age-keygen -o \"$key_file\""
    fi
    if command -v sops >/dev/null 2>&1 && [[ -f "$key_file" ]]; then
        if sops -d "$repo_root/secrets/default.yaml" >/dev/null 2>&1; then
            say "SOPS: secrets/default.yaml decrypts successfully."
        else
            say "SOPS: secrets/default.yaml cannot be decrypted with this user's key."
            say "Re-encrypt it with your age recipient before applying the configuration."
        fi
    else
        say "SOPS: install sops and age, then verify secrets/default.yaml before rebuilding."
    fi
}
