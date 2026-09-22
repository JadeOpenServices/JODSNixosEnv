package graphics

import "testing"

func TestParseHybridNvidiaAndAMD(t *testing.T) {
	result := Parse(`0000:05:00.0 VGA compatible controller [0300]: NVIDIA Corporation AD107M [GeForce RTX 4060 Max-Q / Mobile] [10de:28a0] (rev a1)
0000:c1:00.0 VGA compatible controller [0300]: Advanced Micro Devices, Inc. [AMD/ATI] Phoenix1 [Radeon 780M] [1002:15bf] (rev c8)`)
	if result.Vendor != "nvidia" || result.Type != "hybrid" || !result.Compute {
		t.Fatalf("unexpected topology: %#v", result)
	}
	if result.BusID != "PCI:5:0:0" || result.IntegratedBusID != "PCI:193:0:0" {
		t.Fatalf("unexpected bus IDs: %#v", result)
	}
}

func TestParseIntelOnly(t *testing.T) {
	result := Parse(`0000:00:02.0 VGA compatible controller [0300]: Intel Corporation Meteor Lake-P [Intel Arc Graphics] [8086:7d55] (rev 08)`)
	if result.Vendor != "intel" || result.Type != "integrated" || result.Compute || result.BusID != "PCI:0:2:0" {
		t.Fatalf("unexpected Intel result: %#v", result)
	}
}

func TestParseSelectsNVIDIADeviceID(t *testing.T) {
	input := `0000:00:02.0 VGA compatible controller [0300]: Intel Corporation HD Graphics 620 [8086:5916]
0000:01:00.0 3D controller [0302]: NVIDIA Corporation GM107GLM [Quadro M620] [10de:13b4]
`

	result := Parse(input)

	if result.Vendor != "nvidia" {
		t.Fatalf("Vendor = %q, want nvidia", result.Vendor)
	}
	if result.DeviceID != "13b4" {
		t.Fatalf("DeviceID = %q, want 13b4", result.DeviceID)
	}
	if result.Type != "hybrid" {
		t.Fatalf("Type = %q, want hybrid", result.Type)
	}
	if !result.Compute {
		t.Fatal("Quadro M620 must be classified as compute-capable")
	}
	if result.BusID != "PCI:1:0:0" {
		t.Fatalf("BusID = %q, want PCI:1:0:0", result.BusID)
	}
	if result.IntegratedBusID != "PCI:0:2:0" {
		t.Fatalf(
			"IntegratedBusID = %q, want PCI:0:2:0",
			result.IntegratedBusID,
		)
	}
}

func TestParseFramework13AMD(t *testing.T) {
	result := Parse(`0000:c1:00.0 VGA compatible controller [0300]: Advanced Micro Devices, Inc. [AMD/ATI] Phoenix1 [Radeon 780M] [1002:15bf] (rev c8)`)

	if result.Vendor != "amd" {
		t.Fatalf("Vendor = %q, want amd", result.Vendor)
	}
	if result.DeviceID != "15bf" {
		t.Fatalf("DeviceID = %q, want 15bf", result.DeviceID)
	}
	if result.Type != "integrated" {
		t.Fatalf("Type = %q, want integrated", result.Type)
	}
	if !result.Compute {
		t.Fatal("Phoenix1 must be compute-capable")
	}
	if result.BusID != "PCI:193:0:0" {
		t.Fatalf("BusID = %q, want PCI:193:0:0", result.BusID)
	}
	if result.IntegratedBusID != "PCI:193:0:0" {
		t.Fatalf("IntegratedBusID = %q, want PCI:193:0:0", result.IntegratedBusID)
	}
}
