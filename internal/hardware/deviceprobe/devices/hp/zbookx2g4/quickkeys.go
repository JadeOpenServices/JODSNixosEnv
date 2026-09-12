package zbookx2g4

import "fmt"

const (
	QuickKeysVendorID  = "03f0"
	QuickKeysProductID = "0a56"
)

type QuickKeysEvent struct {
	Button       int  `json:"button"`
	Preset       int  `json:"preset"`
	Pressed      bool `json:"pressed"`
	PresetSwitch bool `json:"presetSwitch"`
}

func DecodeQuickKeysReport(report []byte) (QuickKeysEvent, error) {
	if len(report) != 6 {
		return QuickKeysEvent{}, fmt.Errorf(
			"quick keys report length %d, want 6",
			len(report),
		)
	}

	if report[0] != 0x02 {
		return QuickKeysEvent{}, fmt.Errorf(
			"quick keys report id %#02x, want 0x02",
			report[0],
		)
	}

	if report[1] != 0x00 || report[2] != 0x00 || report[5] != 0x01 {
		return QuickKeysEvent{}, fmt.Errorf(
			"unexpected quick keys framing %x",
			report,
		)
	}

	preset, ok := quickKeysPreset(report[4])
	if !ok {
		return QuickKeysEvent{}, fmt.Errorf(
			"unknown quick keys preset byte %#02x",
			report[4],
		)
	}

	if report[3] == 0 {
		return QuickKeysEvent{
			Preset: preset,
		}, nil
	}

	button, ok := quickKeysButton(report[3])
	if !ok {
		return QuickKeysEvent{}, fmt.Errorf(
			"unknown quick keys button byte %#02x",
			report[3],
		)
	}

	return QuickKeysEvent{
		Button:       button,
		Preset:       preset,
		Pressed:      true,
		PresetSwitch: button == 3,
	}, nil
}

func quickKeysButton(value byte) (int, bool) {
	switch value {
	case 0x01:
		return 1, true
	case 0x02:
		return 2, true
	case 0x04:
		return 3, true
	case 0x08:
		return 4, true
	case 0x10:
		return 5, true
	case 0x20:
		return 6, true
	default:
		return 0, false
	}
}

func quickKeysPreset(value byte) (int, bool) {
	switch value {
	case 0x01:
		return 1, true
	case 0x02:
		return 2, true
	case 0x04:
		return 3, true
	default:
		return 0, false
	}
}
