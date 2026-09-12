package zbookx2g4

import (
	"testing"

	"github.com/bakanura/gjallarOS/internal/hardware/deviceprobe"
)

func TestEvaluateKnownZBookBoundary(t *testing.T) {
	report := Evaluate(deviceprobe.Snapshot{
		SysVendor:   "HP",
		ProductName: "HP ZBook x2 G4",
		BoardName:   "824C",
		PCIDevices: []deviceprobe.PCIDevice{
			{
				Address: "0000:01:00.0",
				Vendor:  "0x10de",
				Device:  "0x13b4",
				Driver:  "nvidia",
			},
		},
		Input: []deviceprobe.InputDevice{
			{Event: "event0", Name: "test input"},
		},
		IIO: []deviceprobe.IIODevice{
			{Device: "iio:device0", Name: "test accelerometer"},
		},
	})

	if len(report.Results) != 11 {
		t.Fatalf("got %d results, want 11", len(report.Results))
	}

	if report.Results[0].Status != StatusPass {
		t.Fatalf("identity did not pass: %+v", report.Results[0])
	}

	if report.Results[1].Status != StatusPass {
		t.Fatalf("graphics did not pass: %+v", report.Results[1])
	}
}

func TestEvaluateRejectsOtherMachine(t *testing.T) {
	report := Evaluate(deviceprobe.Snapshot{
		SysVendor:   "Framework",
		ProductName: "Laptop 13",
		BoardName:   "FRANMDCP07",
	})

	if report.Results[0].Status != StatusFail {
		t.Fatalf("other machine accepted: %+v", report.Results[0])
	}
}

func TestFingerprintPassesWithUpstreamReader(t *testing.T) {
	result := fingerprint(deviceprobe.Snapshot{
		Fingerprint: deviceprobe.FingerprintState{
			Available: true,
			Devices: []deviceprobe.FingerprintDevice{
				{
					Path: "/net/reactivated/Fprint/Device/0",
					Name: "Test Reader",
				},
			},
		},
	})

	if result.Status != StatusPass {
		t.Fatalf("fingerprint did not pass: %+v", result)
	}
}

func TestFingerprintWarnsWhenUpstreamUnavailable(t *testing.T) {
	result := fingerprint(deviceprobe.Snapshot{
		Fingerprint: deviceprobe.FingerprintState{
			Available: false,
			Devices:   []deviceprobe.FingerprintDevice{},
			Warning:   "fprintd unavailable",
		},
	})

	if result.Status != StatusWarn {
		t.Fatalf("fingerprint did not warn: %+v", result)
	}
}

func TestFingerprintWarnsWhenNoReaderFound(t *testing.T) {
	result := fingerprint(deviceprobe.Snapshot{
		Fingerprint: deviceprobe.FingerprintState{
			Available: true,
			Devices:   []deviceprobe.FingerprintDevice{},
		},
	})

	if result.Status != StatusWarn {
		t.Fatalf("fingerprint did not warn: %+v", result)
	}
}

func TestThunderboltPassesWhenObserved(t *testing.T) {
	result := thunderbolt(deviceprobe.Snapshot{
		Thunderbolt: []deviceprobe.ThunderboltDevice{
			{
				Path:       "0-1",
				DeviceName: "Test Dock",
			},
		},
	})

	if result.Status != StatusPass {
		t.Fatalf("thunderbolt did not pass: %+v", result)
	}
}

func TestAudioPassesWhenObserved(t *testing.T) {
	result := audio(deviceprobe.Snapshot{
		Audio: []deviceprobe.AudioDevice{
			{
				Card:   "card0",
				ID:     "Generic",
				Driver: "snd_hda_intel",
			},
		},
	})

	if result.Status != StatusPass {
		t.Fatalf("audio did not pass: %+v", result)
	}
}

func TestCardReaderPassesForRemovableBlockDevice(t *testing.T) {
	result := cardReader(deviceprobe.Snapshot{
		Block: []deviceprobe.BlockDevice{
			{
				Name:      "mmcblk0",
				Removable: true,
				Driver:    "sdhci-pci",
			},
		},
	})

	if result.Status != StatusPass {
		t.Fatalf("card reader did not pass: %+v", result)
	}
}

func TestZBookInputRoleGates(t *testing.T) {
	snapshot := deviceprobe.Snapshot{
		HIDDevices: []deviceprobe.HIDDevice{
			{
				Vendor:  "03eb",
				Product: "8abb",
			},
			{
				Vendor:  "056a",
				Product: "016c",
			},
		},
		Input: []deviceprobe.InputDevice{
			{
				Event: "event1",
				Name:  "Touch",
				Classification: deviceprobe.InputClassification{
					Touchscreen: true,
				},
			},
			{
				Event: "event2",
				Name:  "Pen",
				Classification: deviceprobe.InputClassification{
					Pen: true,
				},
			},
			{
				Event: "event3",
				Name:  "Keyboard",
				Classification: deviceprobe.InputClassification{
					Keyboard: true,
				},
			},
			{
				Event: "event4",
				Name:  "Aux Buttons",
				Classification: deviceprobe.InputClassification{
					Buttons: true,
				},
			},
		},
	}

	if result := touch(snapshot); result.Status != StatusPass {
		t.Fatalf("touch did not pass: %+v", result)
	}
	if result := pen(snapshot); result.Status != StatusPass {
		t.Fatalf("pen did not pass: %+v", result)
	}
	if result := keyboard(snapshot); result.Status != StatusPass {
		t.Fatalf("keyboard did not pass: %+v", result)
	}
	if result := quickKeys(snapshot); result.Status != StatusWarn {
		t.Fatalf("quick-keys candidate should remain WARN before real-device characterization: %+v", result)
	}
}

func TestQuickKeysExactHIDGate(t *testing.T) {
	snapshot := deviceprobe.Snapshot{
		HIDDevices: []deviceprobe.HIDDevice{
			{
				Path:    "0003:03F0:0A56.0001",
				Bus:     "0003",
				Vendor:  "03f0",
				Product: "0a56",
				Name:    "HP Inc. HP ZBook Create x2 Quick Keys",
				Driver:  "hid-generic",
				Hidraw:  []string{"hidraw0"},
			},
		},
	}

	result := quickKeys(snapshot)
	if result.Status != StatusPass {
		t.Fatalf("Quick Keys gate did not pass: %+v", result)
	}
}

func TestMatchesDeviceRequiresExactZBookIdentity(t *testing.T) {
	if !MatchesDevice(deviceprobe.Snapshot{
		SysVendor:   "HP",
		ProductName: "HP ZBook x2 G4",
		BoardName:   "824C",
	}) {
		t.Fatal("exact HP ZBook x2 G4 identity did not match")
	}

	if MatchesDevice(deviceprobe.Snapshot{
		SysVendor:   "Framework",
		ProductName: "Laptop 13 (AMD Ryzen 7040Series)",
		BoardName:   "FRANMDCP07",
	}) {
		t.Fatal("non-ZBook identity incorrectly matched")
	}
}

func TestTouchRequiresZBookHIDAndTouchInput(t *testing.T) {
	snapshot := deviceprobe.Snapshot{
		HIDDevices: []deviceprobe.HIDDevice{
			{
				Vendor:  "03eb",
				Product: "8abb",
			},
		},
		Input: []deviceprobe.InputDevice{
			{
				Event: "event18",
				Name:  "ATML1000:00 03EB:8ABB",
				Classification: deviceprobe.InputClassification{
					Touchscreen: true,
				},
			},
		},
	}

	result := touch(snapshot)
	if result.Status != StatusPass {
		t.Fatalf("touch gate did not pass: %+v", result)
	}
}

func TestTouchRejectsGenericTouchscreenWithoutZBookHID(t *testing.T) {
	snapshot := deviceprobe.Snapshot{
		Input: []deviceprobe.InputDevice{
			{
				Event: "event7",
				Name:  "Generic Touchscreen",
				Classification: deviceprobe.InputClassification{
					Touchscreen: true,
				},
			},
		},
	}

	result := touch(snapshot)
	if result.Status != StatusWarn {
		t.Fatalf("generic touchscreen unexpectedly passed: %+v", result)
	}
}

func TestPenRequiresZBookWacomHIDAndPenInput(t *testing.T) {
	snapshot := deviceprobe.Snapshot{
		HIDDevices: []deviceprobe.HIDDevice{
			{
				Vendor:  "056a",
				Product: "016c",
			},
		},
		Input: []deviceprobe.InputDevice{
			{
				Event: "event21",
				Name:  "WCOM002E:00 056A:016C Stylus",
				Classification: deviceprobe.InputClassification{
					Pen: true,
				},
			},
		},
	}

	result := pen(snapshot)
	if result.Status != StatusPass {
		t.Fatalf("pen gate did not pass: %+v", result)
	}
}

func TestPenRejectsGenericPenWithoutZBookWacomHID(t *testing.T) {
	snapshot := deviceprobe.Snapshot{
		Input: []deviceprobe.InputDevice{
			{
				Event: "event5",
				Name:  "Generic Pen",
				Classification: deviceprobe.InputClassification{
					Pen: true,
				},
			},
		},
	}

	result := pen(snapshot)
	if result.Status != StatusWarn {
		t.Fatalf("generic pen unexpectedly passed: %+v", result)
	}
}
