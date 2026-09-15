package deviceprobe

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type PowerSupply struct {
	Name                        string `json:"name"`
	Type                        string `json:"type,omitempty"`
	Status                      string `json:"status,omitempty"`
	Capacity                    string `json:"capacity,omitempty"`
	ChargeControlStartThreshold string `json:"chargeControlStartThreshold,omitempty"`
	ChargeControlEndThreshold   string `json:"chargeControlEndThreshold,omitempty"`
	ChargeStartThreshold        string `json:"chargeStartThreshold,omitempty"`
	ChargeStopThreshold         string `json:"chargeStopThreshold,omitempty"`
}

type BacklightDevice struct {
	Name             string `json:"name"`
	Brightness       string `json:"brightness,omitempty"`
	ActualBrightness string `json:"actualBrightness,omitempty"`
	MaxBrightness    string `json:"maxBrightness,omitempty"`
}

type VideoDevice struct {
	Device string `json:"device"`
	Name   string `json:"name,omitempty"`
}

type EmbeddedControllerState struct {
	Present bool   `json:"present"`
	Name    string `json:"name,omitempty"`
}

type FanDevice struct {
	HWMon  string `json:"hwmon"`
	Name   string `json:"name,omitempty"`
	Input  string `json:"input,omitempty"`
	Target string `json:"target,omitempty"`
}

type AMDPowerState struct {
	PStateStatus           string   `json:"pstateStatus,omitempty"`
	PlatformProfile        string   `json:"platformProfile,omitempty"`
	PlatformProfileChoices []string `json:"platformProfileChoices"`
}

func powerSupplies(sysRoot string) []PowerSupply {
	paths, _ := filepath.Glob(filepath.Join(sysRoot, "class/power_supply/*"))
	out := make([]PowerSupply, 0, len(paths))

	for _, path := range paths {
		out = append(out, PowerSupply{
			Name:                        filepath.Base(path),
			Type:                        read(filepath.Join(path, "type")),
			Status:                      read(filepath.Join(path, "status")),
			Capacity:                    read(filepath.Join(path, "capacity")),
			ChargeControlStartThreshold: read(filepath.Join(path, "charge_control_start_threshold")),
			ChargeControlEndThreshold:   read(filepath.Join(path, "charge_control_end_threshold")),
			ChargeStartThreshold:        read(filepath.Join(path, "charge_start_threshold")),
			ChargeStopThreshold:         read(filepath.Join(path, "charge_stop_threshold")),
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func backlightDevices(sysRoot string) []BacklightDevice {
	paths, _ := filepath.Glob(filepath.Join(sysRoot, "class/backlight/*"))
	out := make([]BacklightDevice, 0, len(paths))

	for _, path := range paths {
		out = append(out, BacklightDevice{
			Name:             filepath.Base(path),
			Brightness:       read(filepath.Join(path, "brightness")),
			ActualBrightness: read(filepath.Join(path, "actual_brightness")),
			MaxBrightness:    read(filepath.Join(path, "max_brightness")),
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func videoDevices(sysRoot string) []VideoDevice {
	paths, _ := filepath.Glob(filepath.Join(sysRoot, "class/video4linux/*"))
	out := make([]VideoDevice, 0, len(paths))

	for _, path := range paths {
		out = append(out, VideoDevice{
			Device: filepath.Base(path),
			Name:   read(filepath.Join(path, "name")),
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Device < out[j].Device })
	return out
}

func embeddedControllerState(sysRoot string) EmbeddedControllerState {
	path := filepath.Join(sysRoot, "class/chromeos/cros_ec")
	if _, err := os.Stat(path); err != nil {
		return EmbeddedControllerState{}
	}

	return EmbeddedControllerState{
		Present: true,
		Name:    "cros_ec",
	}
}

func fanDevices(sysRoot string) []FanDevice {
	paths, _ := filepath.Glob(filepath.Join(sysRoot, "class/hwmon/hwmon*"))
	out := []FanDevice{}

	for _, path := range paths {
		name := read(filepath.Join(path, "name"))
		if !strings.EqualFold(name, "cros_ec") {
			continue
		}

		fans, _ := filepath.Glob(filepath.Join(path, "fan*_input"))
		for _, inputPath := range fans {
			base := strings.TrimSuffix(filepath.Base(inputPath), "_input")
			out = append(out, FanDevice{
				HWMon:  filepath.Base(path),
				Name:   name,
				Input:  read(inputPath),
				Target: read(filepath.Join(path, base+"_target")),
			})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].HWMon == out[j].HWMon {
			return out[i].Input < out[j].Input
		}
		return out[i].HWMon < out[j].HWMon
	})
	return out
}

func amdPowerState(sysRoot string) AMDPowerState {
	choices := strings.Fields(read(filepath.Join(
		sysRoot,
		"firmware/acpi/platform_profile_choices",
	)))
	if choices == nil {
		choices = []string{}
	}

	return AMDPowerState{
		PStateStatus: read(filepath.Join(
			sysRoot,
			"devices/system/cpu/amd_pstate/status",
		)),
		PlatformProfile: read(filepath.Join(
			sysRoot,
			"firmware/acpi/platform_profile",
		)),
		PlatformProfileChoices: choices,
	}
}
