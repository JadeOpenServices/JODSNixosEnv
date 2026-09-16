package orientation

import "strings"

func IsSensor(name string, channels []string) bool {
	lowerName := strings.ToLower(name)

	if strings.Contains(lowerName, "accel") ||
		strings.Contains(lowerName, "orientation") ||
		strings.Contains(lowerName, "rotation") {
		return true
	}

	for _, channel := range channels {
		if strings.HasPrefix(strings.ToLower(channel), "in_accel_") {
			return true
		}
	}

	return false
}
