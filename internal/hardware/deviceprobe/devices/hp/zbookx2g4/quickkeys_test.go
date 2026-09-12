package zbookx2g4

import "testing"

func TestDecodeQuickKeysActionButtons(t *testing.T) {
	tests := []struct {
		name   string
		report []byte
		button int
		preset int
	}{
		{"button1 preset3", []byte{0x02, 0, 0, 0x01, 0x04, 0x01}, 1, 3},
		{"button2 preset3", []byte{0x02, 0, 0, 0x02, 0x04, 0x01}, 2, 3},
		{"button4 preset3", []byte{0x02, 0, 0, 0x08, 0x04, 0x01}, 4, 3},
		{"button5 preset2", []byte{0x02, 0, 0, 0x10, 0x02, 0x01}, 5, 2},
		{"button6 preset2", []byte{0x02, 0, 0, 0x20, 0x02, 0x01}, 6, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := DecodeQuickKeysReport(tt.report)
			if err != nil {
				t.Fatal(err)
			}

			if !event.Pressed {
				t.Fatal("press report decoded as released")
			}
			if event.Button != tt.button {
				t.Fatalf("button=%d, want %d", event.Button, tt.button)
			}
			if event.Preset != tt.preset {
				t.Fatalf("preset=%d, want %d", event.Preset, tt.preset)
			}
			if event.PresetSwitch {
				t.Fatal("action button marked as preset switch")
			}
		})
	}
}

func TestDecodeQuickKeysPresetSwitch(t *testing.T) {
	tests := []struct {
		report []byte
		preset int
	}{
		{[]byte{0x02, 0, 0, 0x04, 0x01, 0x01}, 1},
		{[]byte{0x02, 0, 0, 0x04, 0x02, 0x01}, 2},
		{[]byte{0x02, 0, 0, 0x04, 0x04, 0x01}, 3},
	}

	for _, tt := range tests {
		event, err := DecodeQuickKeysReport(tt.report)
		if err != nil {
			t.Fatal(err)
		}

		if event.Button != 3 || !event.Pressed || !event.PresetSwitch {
			t.Fatalf("unexpected preset switch event: %+v", event)
		}
		if event.Preset != tt.preset {
			t.Fatalf("preset=%d, want %d", event.Preset, tt.preset)
		}
	}
}

func TestDecodeQuickKeysRelease(t *testing.T) {
	event, err := DecodeQuickKeysReport(
		[]byte{0x02, 0, 0, 0x00, 0x02, 0x01},
	)
	if err != nil {
		t.Fatal(err)
	}

	if event.Pressed {
		t.Fatal("release report decoded as pressed")
	}
	if event.Button != 0 {
		t.Fatalf("release button=%d, want 0", event.Button)
	}
	if event.Preset != 2 {
		t.Fatalf("preset=%d, want 2", event.Preset)
	}
}

func TestDecodeQuickKeysRejectsUnknownReports(t *testing.T) {
	tests := [][]byte{
		{0x02},
		{0x01, 0, 0, 0x01, 0x01, 0x01},
		{0x02, 1, 0, 0x01, 0x01, 0x01},
		{0x02, 0, 0, 0x40, 0x01, 0x01},
		{0x02, 0, 0, 0x01, 0x08, 0x01},
		{0x02, 0, 0, 0x01, 0x01, 0x00},
	}

	for _, report := range tests {
		if _, err := DecodeQuickKeysReport(report); err == nil {
			t.Fatalf("accepted invalid report %x", report)
		}
	}
}
