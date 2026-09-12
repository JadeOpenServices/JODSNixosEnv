package inputclass

import (
	"strings"
	"testing"
)

func TestPenCapabilityClassification(t *testing.T) {
	device := Device{
		Name: "Wacom Tablet",
		Key:  "1" + strings.Repeat("0", 80),
		Abs:  "3",
	}

	if !IsPen(device) {
		t.Fatal("pen capability device not classified as pen")
	}
}

func TestTouchscreenCapabilityClassification(t *testing.T) {
	device := Device{
		Name:       "ELAN Touch Device",
		Properties: "2",
		Abs:        "60000000000000",
	}

	if !IsTouchscreen(device) {
		t.Fatal("direct multitouch device not classified as touchscreen")
	}
}

func TestTouchpadIsNotTouchscreen(t *testing.T) {
	device := Device{
		Name:       "Touchpad",
		Properties: "5",
		Abs:        "60000000000000",
	}

	if IsTouchscreen(device) {
		t.Fatal("touchpad classified as touchscreen")
	}
}

func TestKeyboardClassification(t *testing.T) {
	device := Device{
		Key: "2000000000000 1000000000000 40000000",
	}

	// Use explicit capability construction instead of depending on a device name.
	device.Key = setBits(30, 44, 57)

	if !IsKeyboard(device) {
		t.Fatal("keyboard capability device not classified as keyboard")
	}
}

func TestButtonInputClassification(t *testing.T) {
	device := Device{
		Key: setBits(115),
	}

	if !IsButtonInput(device) {
		t.Fatal("auxiliary button device not classified as buttons")
	}
}

func setBits(bits ...uint) string {
	if len(bits) == 0 {
		return "0"
	}

	max := uint(0)
	for _, bit := range bits {
		if bit > max {
			max = bit
		}
	}

	nibbles := int(max/4) + 1
	hex := make([]byte, nibbles)
	for i := range hex {
		hex[i] = '0'
	}

	for _, bit := range bits {
		index := nibbles - 1 - int(bit/4)
		value := byte(1 << (bit % 4))

		current := hex[index]
		var nibble byte
		switch {
		case current >= '0' && current <= '9':
			nibble = current - '0'
		case current >= 'a' && current <= 'f':
			nibble = current - 'a' + 10
		}

		nibble |= value
		if nibble < 10 {
			hex[index] = '0' + nibble
		} else {
			hex[index] = 'a' + nibble - 10
		}
	}

	return string(hex)
}
