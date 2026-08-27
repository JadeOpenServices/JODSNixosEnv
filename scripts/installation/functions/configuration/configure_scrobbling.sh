configure_scrobbling() {
    local repo_root="$1" username="$2" answer token service_username recipient key_file plain_file home_dir
    cfg_enable_scrobbling=false
    cfg_enable_lastfm=false
    cfg_enable_listenbrainz=false
    cfg_lastfm_username=''
    cfg_listenbrainz_username=''

    read -r -p 'Enable Last.fm and/or ListenBrainz scrobbling? [y/N] ' answer
    [[ "$answer" =~ ^([yY]|[yY][eE][sS])$ ]] || return 0

    require_command sops
    require_command age-keygen
    home_dir="$(user_home_dir "$username")"
    key_file="$home_dir/.config/sops/age/keys.txt"
    if [[ ! -f "$key_file" ]]; then
        mkdir -p "$(dirname "$key_file")"
        age-keygen -o "$key_file" >/dev/null
        chmod 600 "$key_file"
        say "Generated an age key at $key_file. Back it up securely."
    fi
    recipient="$(awk '/public key:/ {print $NF; exit}' "$key_file")"
    [[ "$recipient" == age1* ]] || die "Could not read the public recipient from $key_file"

    read -r -p 'Enable Last.fm? [y/N] ' answer
    if [[ "$answer" =~ ^([yY]|[yY][eE][sS])$ ]]; then
        read -r -p 'Last.fm username: ' cfg_lastfm_username
        read -r -s -p 'Last.fm token/password: ' token; printf '\n'
        cfg_enable_lastfm=true
        lastfm_token="$token"
        unset token
    fi
    read -r -p 'Enable ListenBrainz? [y/N] ' answer
    if [[ "$answer" =~ ^([yY]|[yY][eE][sS])$ ]]; then
        read -r -p 'ListenBrainz username: ' cfg_listenbrainz_username
        read -r -s -p 'ListenBrainz token: ' token; printf '\n'
        cfg_enable_listenbrainz=true
        listenbrainz_token="$token"
        unset token
    fi
    [[ "$cfg_enable_lastfm" == true || "$cfg_enable_listenbrainz" == true ]] || return 0
    cfg_enable_scrobbling=true

    plain_file="$(mktemp)"
    {
        if [[ "$cfg_enable_lastfm" == true ]]; then
            printf "lastfm: '%s'\n" "${lastfm_token//\'/\'\'}"
        fi
        if [[ "$cfg_enable_listenbrainz" == true ]]; then
            printf "listenbrainz: '%s'\n" "${listenbrainz_token//\'/\'\'}"
        fi
    } > "$plain_file"
    if ! sops --encrypt --age "$recipient" --output "$repo_root/secrets/default.yaml" "$plain_file"; then
        rm -f -- "$plain_file"
        unset lastfm_token listenbrainz_token
        die 'Could not encrypt scrobbling credentials; no credentials were written.'
    fi
    rm -f -- "$plain_file"
    printf '%s\n' \
        '# Managed by scripts/installation/install.sh.' \
        'creation_rules:' \
        '  - path_regex: secrets/[^/]+\\.(yaml|json|env|ini)$' \
        "    age: $recipient" > "$repo_root/.sops.yaml"
    unset lastfm_token listenbrainz_token
    say 'Encrypted scrobbling credentials in secrets/default.yaml.'
}
