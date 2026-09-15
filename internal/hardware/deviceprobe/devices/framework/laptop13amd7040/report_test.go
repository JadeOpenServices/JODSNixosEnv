package laptop13amd7040

import (
	"testing"

	"github.com/bakanura/gjallarOS/internal/hardware/deviceprobe"
)

func realBoundary() deviceprobe.Snapshot {
	return deviceprobe.Snapshot{
		SysVendor:   "Framework",
		ProductName: "Laptop 13 (AMD Ryzen 7040Series)",
		BoardName:   "FRANMDCP07",
		PCIDevices: []deviceprobe.PCIDevice{
			{Vendor: "0x1002", Device: "0x15bf", Driver: "amdgpu"},
			{Vendor: "0x10ec", Device: "0xb852", Driver: "rtw89_8852be"},
			{Vendor: "0x1022", Device: "0x15c7", Driver: "ccp"},
		},
		USBDevices: []deviceprobe.USBDevice{
			{Vendor: "27c6", Product: "609c"},
			{Vendor: "13d3", Product: "3571"},
		},
		HIDDevices: []deviceprobe.HIDDevice{
			{Vendor: "093a", Product: "0274", Driver: "hid-multitouch"},
		},
		IIO:             []deviceprobe.IIODevice{{Device: "iio:device0", Name: "als"}},
		ThunderboltHost: true,
		Thunderbolt: []deviceprobe.ThunderboltDevice{
			{Path: "0-0", Generation: "4", Authorized: "1"},
			{Path: "1-0", Generation: "4", Authorized: "1"},
		},
		Audio: []deviceprobe.AudioDevice{
			{Card: "card0", Driver: "snd_hda_intel"},
			{Card: "card1", Driver: "snd_hda_intel"},
		},
		PowerSupplies: []deviceprobe.PowerSupply{
			{Name: "BAT1", Type: "Battery", Status: "Discharging", Capacity: "58"},
		},
		Backlights: []deviceprobe.BacklightDevice{
			{Name: "amdgpu_bl1", Brightness: "65535", ActualBrightness: "62579", MaxBrightness: "65535"},
		},
		VideoDevices: []deviceprobe.VideoDevice{},
		EmbeddedControl: deviceprobe.EmbeddedControllerState{
			Present: true,
			Name:    "cros_ec",
		},
		Fans: []deviceprobe.FanDevice{
			{HWMon: "hwmon7", Name: "cros_ec", Input: "2925", Target: "6182"},
		},
		AMDPower: deviceprobe.AMDPowerState{
			PStateStatus:           "active",
			PlatformProfile:        "performance",
			PlatformProfileChoices: []string{"low-power", "balanced", "performance"},
		},
		Fingerprint: deviceprobe.FingerprintState{
			Available: true,
			Devices: []deviceprobe.FingerprintDevice{
				{Name: "Goodix MOC Fingerprint Sensor"},
			},
		},
	}
}

func TestEvaluateRealBoundary(t *testing.T) {
	report := Evaluate(realBoundary())
	for _, result := range report.Results {
		if result.Status == StatusFail {
			t.Fatalf("%s failed: %+v", result.Gate, result)
		}
	}
}

func TestRealBoundaryExpectedWarnings(t *testing.T) {
	report := Evaluate(realBoundary())

	warnings := map[string]bool{}
	for _, result := range report.Results {
		if result.Status == StatusWarn {
			warnings[result.Gate] = true
		}
	}

	if !warnings["charge-control"] {
		t.Fatal("missing charge-control warning")
	}
	if !warnings["camera"] {
		t.Fatal("missing camera warning")
	}
}

func TestRejectsOtherMachine(t *testing.T) {
	report := Evaluate(deviceprobe.Snapshot{
		SysVendor:   "HP",
		ProductName: "HP ZBook x2 G4",
		BoardName:   "824C",
	})
	if report.Results[0].Status != StatusFail {
		t.Fatalf("wrong machine accepted: %+v", report.Results[0])
	}
}

func TestGraphicsRequiresAmdgpu(t *testing.T) {
	s := realBoundary()
	s.PCIDevices[0].Driver = ""
	if result := graphics(s); result.Status != StatusFail {
		t.Fatalf("unbound Radeon unexpectedly passed: %+v", result)
	}
}

func TestFingerprintWarnsWithoutFprintd(t *testing.T) {
	s := realBoundary()
	s.Fingerprint.Devices = nil
	if result := fingerprint(s); result.Status != StatusWarn {
		t.Fatalf("fingerprint should warn: %+v", result)
	}
}
