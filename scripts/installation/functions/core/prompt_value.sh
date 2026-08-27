prompt_value() {
    local prompt="$1" default="$2" value
    read -r -p "$prompt [$default]: " value
    REPLY="${value:-$default}"
}
