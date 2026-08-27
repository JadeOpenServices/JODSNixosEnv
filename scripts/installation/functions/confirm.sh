confirm() {
    local answer
    read -r -p "$1 [y/N] " answer
    [[ "$answer" =~ ^([yY]|[yY][eE][sS])$ ]]
}
