say() {
    local message="$*" level='INFO' color='\033[1;36m' reset='\033[0m'
    case "$message" in
        '[WARN]'*) level='WARN'; color='\033[1;33m' ;;
        '[ERROR]'*) level='ERROR'; color='\033[1;31m' ;;
        '[DEBUG]'*) level='DEBUG'; color='\033[0;90m' ;;
        '[NOTE]'*) level='INFO' ;;
    esac
    message="${message#'[WARN] '}"; message="${message#'[ERROR] '}"
    message="${message#'[DEBUG] '}"; message="${message#'[NOTE] '}"
    if [[ -t 1 ]]; then
        printf '%b[%s]%b %s\n' "$color" "$level" "$reset" "$message"
    else
        printf '[%s] %s\n' "$level" "$message"
    fi
}
