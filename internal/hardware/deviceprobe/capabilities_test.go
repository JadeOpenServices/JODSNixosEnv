package deviceprobe

import "testing"

func TestCapabilitiesDistinguishPresentHardwareFromAttachedDevices(t *testing.T) {
	snapshot := Snapshot{
		PCI: []string{
			"0000:04:00.0 PCI bridge: Example Thunderbolt Controller",
			"0000:02:00.0 Unassigned class: Example PCI Express Card Reader",
		},
		USB: []string{
			"Bus 001 Device 005: ID 1234:5678 Example Fingerprint Reader",
		},
		Thunderbolt: []ThunderboltDevice{
			{
				Path:       "0-0",
				DeviceName: "Host",
				Generation: "3",
			},
		},
		Fingerprint: FingerprintState{
			Available: true,
			Devices:   []FingerprintDevice{},
		},
	}

	got := DetectCapabilities(snapshot)

	if !got.Thunderbolt.Present {
		t.Fatal("present Thunderbolt controller/domain reported absent")
	}

	if !got.CardReader.Present {
		t.Fatal("present card-reader controller reported absent")
	}

	if !got.Fingerprint.Present {
		t.Fatal("present fingerprint hardware reported absent")
	}

	if got.Fingerprint.UpstreamSupported {
		t.Fatal("unsupported fingerprint hardware reported upstream-supported")
	}
}

func TestFingerprintCapabilityReportsUpstreamSupportSeparately(t *testing.T) {
	got := DetectCapabilities(Snapshot{
		Fingerprint: FingerprintState{
			Available: true,
			Devices: []FingerprintDevice{
				{Path: "/net/reactivated/Fprint/Device/0"},
			},
		},
	})

	if !got.Fingerprint.Present {
		t.Fatal("fprintd device did not imply fingerprint hardware presence")
	}

	if !got.Fingerprint.UpstreamSupported {
		t.Fatal("fprintd device did not imply upstream support")
	}
}

func TestCapabilitiesDoNotRequireInsertedMediaOrExternalDevices(t *testing.T) {
	got := DetectCapabilities(Snapshot{
		PCI: []string{
			"0000:02:00.0 Unassigned class: Example PCI Express Card Reader",
		},
		Thunderbolt: []ThunderboltDevice{
			{Path: "0-0", DeviceName: "Host"},
		},
		Block: []BlockDevice{},
	})

	if !got.Thunderbolt.Present {
		t.Fatal("idle Thunderbolt controller reported absent")
	}

	if !got.CardReader.Present {
		t.Fatal("empty card reader reported absent")
	}
}

func TestThunderboltHostPersistsWithoutEnumeratedController(t *testing.T) {
	got := DetectCapabilities(Snapshot{
		ThunderboltHost: true,
		Thunderbolt:     []ThunderboltDevice{},
		PCIDevices:      []PCIDevice{},
		PCI:             []string{},
	})

	if !got.Thunderbolt.Present {
		t.Fatal("persistent Thunderbolt host/domain reported absent")
	}
}

func TestDetectCapabilitiesFindsOrientationFromAccelerometer(t *testing.T) {
	got := DetectCapabilities(Snapshot{
		IIO: []IIODevice{
			{
				Device: "iio:device4",
				Name:   "accel_3d",
				Channels: []string{
					"in_accel_x_raw",
					"in_accel_y_raw",
					"in_accel_z_raw",
				},
			},
		},
	})

	if !got.Orientation.Present {
		t.Fatal("accelerometer did not enable orientation capability")
	}
}

func TestDetectCapabilitiesFindsOrientationDevice(t *testing.T) {
	got := DetectCapabilities(Snapshot{
		IIO: []IIODevice{
			{
				Device: "iio:device6",
				Name:   "relative_orientation",
			},
		},
	})

	if !got.Orientation.Present {
		t.Fatal("orientation IIO device was not detected")
	}
}

func TestDetectCapabilitiesDoesNotTreatALSAsOrientation(t *testing.T) {
	got := DetectCapabilities(Snapshot{
		IIO: []IIODevice{
			{
				Device: "iio:device0",
				Name:   "als",
			},
		},
	})

	if got.Orientation.Present {
		t.Fatal("ambient-light sensor incorrectly enabled orientation capability")
	}
}
