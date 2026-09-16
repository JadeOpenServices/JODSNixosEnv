package deviceprobe

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLaptopStateCollectors(t *testing.T) {
	root := t.TempDir()

	battery := filepath.Join(root, "class/power_supply/BAT1")
	writeProbeFile(t, filepath.Join(battery, "type"), "Battery")
	writeProbeFile(t, filepath.Join(battery, "status"), "Discharging")
	writeProbeFile(t, filepath.Join(battery, "capacity"), "58")
	writeProbeFile(t, filepath.Join(battery, "charge_control_end_threshold"), "80")

	backlight := filepath.Join(root, "class/backlight/amdgpu_bl1")
	writeProbeFile(t, filepath.Join(backlight, "brightness"), "40000")
	writeProbeFile(t, filepath.Join(backlight, "actual_brightness"), "39000")
	writeProbeFile(t, filepath.Join(backlight, "max_brightness"), "65535")

	video := filepath.Join(root, "class/video4linux/video0")
	writeProbeFile(t, filepath.Join(video, "name"), "Framework Laptop Webcam")

	if err := os.MkdirAll(filepath.Join(root, "class/chromeos/cros_ec"), 0755); err != nil {
		t.Fatal(err)
	}

	hwmon := filepath.Join(root, "class/hwmon/hwmon7")
	writeProbeFile(t, filepath.Join(hwmon, "name"), "cros_ec")
	writeProbeFile(t, filepath.Join(hwmon, "fan1_input"), "2925")
	writeProbeFile(t, filepath.Join(hwmon, "fan1_target"), "6182")

	writeProbeFile(t, filepath.Join(root, "devices/system/cpu/amd_pstate/status"), "active")
	writeProbeFile(t, filepath.Join(root, "firmware/acpi/platform_profile"), "performance")
	writeProbeFile(t, filepath.Join(root, "firmware/acpi/platform_profile_choices"), "low-power balanced performance")

	power := powerSupplies(root)
	if len(power) != 1 ||
		power[0].Name != "BAT1" ||
		power[0].Capacity != "58" ||
		power[0].ChargeControlEndThreshold != "80" {
		t.Fatalf("unexpected power supply: %+v", power)
	}

	backlights := backlightDevices(root)
	if len(backlights) != 1 ||
		backlights[0].Name != "amdgpu_bl1" ||
		backlights[0].MaxBrightness != "65535" {
		t.Fatalf("unexpected backlight: %+v", backlights)
	}

	videoDevices := videoDevices(root)
	if len(videoDevices) != 1 || videoDevices[0].Device != "video0" {
		t.Fatalf("unexpected video devices: %+v", videoDevices)
	}

	ec := embeddedControllerState(root)
	if !ec.Present || ec.Name != "cros_ec" {
		t.Fatalf("unexpected EC state: %+v", ec)
	}

	fans := fanDevices(root)
	if len(fans) != 1 || fans[0].Input != "2925" || fans[0].Target != "6182" {
		t.Fatalf("unexpected fans: %+v", fans)
	}

	amd := amdPowerState(root)
	if amd.PStateStatus != "active" ||
		amd.PlatformProfile != "performance" ||
		len(amd.PlatformProfileChoices) != 3 {
		t.Fatalf("unexpected AMD power state: %+v", amd)
	}
}
