package inputclass

import "strings"

type Device struct {
	Name       string
	Properties string
	Key        string
	Abs        string
}

func IsPen(device Device) bool {
	name := strings.ToLower(device.Name)

	if strings.Contains(name, "stylus") ||
		strings.Contains(name, " pen") {
		return true
	}

	return CapabilityBit(device.Key, 320) &&
		CapabilityBit(device.Abs, 0) &&
		CapabilityBit(device.Abs, 1)
}

func IsKeyboard(device Device) bool {
	return CapabilityBit(device.Key, 30) && // KEY_A
		CapabilityBit(device.Key, 44) && // KEY_Z
		CapabilityBit(device.Key, 57) // KEY_SPACE
}

func IsButtonInput(device Device) bool {
	if IsKeyboard(device) || IsPen(device) || IsTouchscreen(device) {
		return false
	}

	for _, bit := range []uint{
		28,  // KEY_ENTER
		103, // KEY_UP
		108, // KEY_DOWN
		105, // KEY_LEFT
		106, // KEY_RIGHT
		113, // KEY_MUTE
		114, // KEY_VOLUMEDOWN
		115, // KEY_VOLUMEUP
	} {
		if CapabilityBit(device.Key, bit) {
			return true
		}
	}

	return false
}

func IsTouchscreen(device Device) bool {
	if strings.Contains(strings.ToLower(device.Name), "touchscreen") {
		return true
	}

	return CapabilityBit(device.Properties, 1) &&
		CapabilityBit(device.Abs, 53) &&
		CapabilityBit(device.Abs, 54)
}

// Linux exposes input capability bitsets as hexadecimal, most-significant
// word first. Reading from the right keeps this independent of word size.
func CapabilityBit(value string, bit uint) bool {
	hex := strings.ReplaceAll(strings.TrimSpace(value), " ", "")
	nibble := int(bit / 4)

	if nibble >= len(hex) {
		return false
	}

	digit := hex[len(hex)-1-nibble]

	var number byte
	switch {
	case digit >= '0' && digit <= '9':
		number = digit - '0'
	case digit >= 'a' && digit <= 'f':
		number = digit - 'a' + 10
	case digit >= 'A' && digit <= 'F':
		number = digit - 'A' + 10
	default:
		return false
	}

	return number&(1<<(bit%4)) != 0
}
