// Package nixrender emits the small Nix literal subset used by installer
// generated settings. It deliberately has no evaluation capability.
package nixrender

import "strings"

// String returns a Nix double-quoted string literal. In addition to ordinary
// quoting it escapes interpolation markers, so a value from user.config.json
// can never become executable Nix during settings rendering.
func String(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	value = strings.ReplaceAll(value, "${", "\\${")
	value = strings.ReplaceAll(value, "\n", "\\n")
	value = strings.ReplaceAll(value, "\r", "\\r")
	return "\"" + value + "\""
}

func Strings(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, String(value))
	}
	return "[ " + strings.Join(quoted, " ") + " ]"
}
