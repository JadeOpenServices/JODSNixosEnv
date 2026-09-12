package deviceprobe

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type HIDDevice struct {
	Path    string   `json:"path"`
	Bus     string   `json:"bus"`
	Vendor  string   `json:"vendor"`
	Product string   `json:"product"`
	Name    string   `json:"name,omitempty"`
	Driver  string   `json:"driver,omitempty"`
	Hidraw  []string `json:"hidraw"`
}

func hidDevices(sysRoot string) []HIDDevice {
	paths, _ := filepath.Glob(filepath.Join(sysRoot, "bus/hid/devices/*"))
	devices := make([]HIDDevice, 0, len(paths))

	for _, path := range paths {
		bus, vendor, product := hidIdentity(filepath.Join(path, "uevent"))
		if vendor == "" || product == "" {
			continue
		}

		devices = append(devices, HIDDevice{
			Path:    filepath.Base(path),
			Bus:     bus,
			Vendor:  vendor,
			Product: product,
			Name:    hidName(filepath.Join(path, "uevent")),
			Driver:  driverName(filepath.Join(path, "driver")),
			Hidraw:  hidrawForDevice(sysRoot, path),
		})
	}

	sort.Slice(devices, func(i, j int) bool {
		return devices[i].Path < devices[j].Path
	})

	return devices
}

func hidIdentity(path string) (string, string, string) {
	for _, line := range strings.Split(read(path), "\n") {
		if !strings.HasPrefix(line, "HID_ID=") {
			continue
		}

		parts := strings.Split(strings.TrimPrefix(line, "HID_ID="), ":")
		if len(parts) != 3 {
			return "", "", ""
		}

		return normalizeHID(parts[0], 4),
			normalizeHID(parts[1], 4),
			normalizeHID(parts[2], 4)
	}

	return "", "", ""
}

func hidName(path string) string {
	for _, line := range strings.Split(read(path), "\n") {
		if strings.HasPrefix(line, "HID_NAME=") {
			return strings.TrimSpace(strings.TrimPrefix(line, "HID_NAME="))
		}
	}
	return ""
}

func normalizeHID(value string, width int) string {
	n, err := strconv.ParseUint(strings.TrimSpace(value), 16, 32)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%0*x", width, n)
}

func hidrawForDevice(sysRoot, target string) []string {
	nodes, _ := filepath.Glob(filepath.Join(sysRoot, "class/hidraw/hidraw*"))
	out := make([]string, 0)

	targetResolved, err := filepath.EvalSymlinks(target)
	if err != nil {
		return out
	}

	for _, node := range nodes {
		deviceResolved, err := filepath.EvalSymlinks(filepath.Join(node, "device"))
		if err != nil {
			continue
		}

		if deviceResolved == targetResolved {
			out = append(out, filepath.Base(node))
		}
	}

	sort.Strings(out)
	return out
}

func nonNilHIDDevices(value []HIDDevice) []HIDDevice {
	if value == nil {
		return []HIDDevice{}
	}
	return value
}

func ensureHIDFixtureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}
