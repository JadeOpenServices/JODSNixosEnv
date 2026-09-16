package deviceprobe

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInputDevices(t *testing.T) {
	root := t.TempDir()
	event := filepath.Join(root, "class/input/event12")

	writeProbeFile(t, filepath.Join(event, "device/name"), "Wacom Pen")
	writeProbeFile(t, filepath.Join(event, "device/capabilities/ev"), "1b")
	writeProbeFile(t, filepath.Join(event, "device/capabilities/key"), "c03")
	writeProbeFile(t, filepath.Join(event, "device/capabilities/abs"), "1000003")

	got := inputDevices(root)
	if len(got) != 1 {
		t.Fatalf("got %d input devices, want 1", len(got))
	}

	if got[0].Event != "event12" ||
		got[0].Name != "Wacom Pen" ||
		got[0].Capabilities.Abs != "1000003" {
		t.Fatalf("unexpected input device: %+v", got[0])
	}
}

func TestIIODevices(t *testing.T) {
	root := t.TempDir()
	device := filepath.Join(root, "bus/iio/devices/iio:device0")

	writeProbeFile(t, filepath.Join(device, "name"), "accel-test")
	writeProbeFile(t, filepath.Join(device, "in_accel_x_raw"), "0")
	writeProbeFile(t, filepath.Join(device, "in_accel_y_raw"), "0")
	writeProbeFile(t, filepath.Join(device, "in_accel_scale"), "1")

	if err := os.WriteFile(
		filepath.Join(device, "sampling_frequency"),
		[]byte("100\n"),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	got := iioDevices(root)
	if len(got) != 1 {
		t.Fatalf("got %d IIO devices, want 1", len(got))
	}

	if got[0].Name != "accel-test" || len(got[0].Channels) != 3 {
		t.Fatalf("unexpected IIO device: %+v", got[0])
	}
}

func TestInputDevicesExposeKeyboardAndButtonClassification(t *testing.T) {
	root := t.TempDir()

	keyboard := filepath.Join(root, "class/input/event20")
	writeProbeFile(t, filepath.Join(keyboard, "device/name"), "Test Keyboard")
	writeProbeFile(
		t,
		filepath.Join(keyboard, "device/capabilities/key"),
		setInputBits(30, 44, 57),
	)

	buttons := filepath.Join(root, "class/input/event21")
	writeProbeFile(t, filepath.Join(buttons, "device/name"), "Test Buttons")
	writeProbeFile(
		t,
		filepath.Join(buttons, "device/capabilities/key"),
		setInputBits(115),
	)

	got := inputDevices(root)
	if len(got) != 2 {
		t.Fatalf("got %d input devices, want 2", len(got))
	}

	if !got[0].Classification.Keyboard {
		t.Fatalf("keyboard classification missing: %+v", got[0])
	}
	if got[0].Classification.Buttons {
		t.Fatalf("keyboard misclassified as auxiliary buttons: %+v", got[0])
	}

	if !got[1].Classification.Buttons {
		t.Fatalf("button classification missing: %+v", got[1])
	}
	if got[1].Classification.Keyboard {
		t.Fatalf("button device misclassified as keyboard: %+v", got[1])
	}
}

func setInputBits(bits ...uint) string {
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
