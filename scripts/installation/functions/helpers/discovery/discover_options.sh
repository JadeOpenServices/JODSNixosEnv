discover_options() {
    local directory="$1" kind="$2" path
    case "$kind" in
        directory)
            find "$directory" -mindepth 1 -maxdepth 1 -type d -printf '%f\n' | sort
            ;;
        nix-file)
            for path in "$directory"/*.nix; do
                [[ -f "$path" ]] && basename "${path%.nix}"
            done | sort
            ;;
        *) die "Unsupported option discovery type: $kind" ;;
    esac
}
