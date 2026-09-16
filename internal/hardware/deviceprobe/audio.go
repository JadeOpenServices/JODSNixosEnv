package deviceprobe

import (
	"os"
	"path/filepath"
	"sort"
)

type AudioDevice struct {
	Card       string `json:"card"`
	ID         string `json:"id,omitempty"`
	Driver     string `json:"driver,omitempty"`
	DevicePath string `json:"devicePath,omitempty"`
}

func audioDevices(sysRoot string) []AudioDevice {
	paths, _ := filepath.Glob(filepath.Join(sysRoot, "class/sound/card*"))
	devices := make([]AudioDevice, 0, len(paths))

	for _, path := range paths {
		card := filepath.Base(path)
		if card == "" {
			continue
		}

		devices = append(devices, AudioDevice{
			Card:       card,
			ID:         read(filepath.Join(path, "id")),
			Driver:     driverName(filepath.Join(path, "device/driver")),
			DevicePath: devicePath(filepath.Join(path, "device")),
		})
	}

	sort.Slice(devices, func(i, j int) bool {
		return devices[i].Card < devices[j].Card
	})

	return devices
}

func devicePath(path string) string {
	target, err := filepath.EvalSymlinks(path)
	if err != nil {
		target, err = os.Readlink(path)
		if err != nil {
			return ""
		}
	}

	return filepath.Base(target)
}
