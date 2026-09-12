package deviceprobe

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type PCIDevice struct {
	Address string `json:"address"`
	Vendor  string `json:"vendor"`
	Device  string `json:"device"`
	Class   string `json:"class"`
	Driver  string `json:"driver,omitempty"`
}

type USBDevice struct {
	Path    string `json:"path"`
	Vendor  string `json:"vendor"`
	Product string `json:"product"`
	Class   string `json:"class"`
	Driver  string `json:"driver,omitempty"`
}

func pciDevices(sysRoot string) []PCIDevice {
	paths, _ := filepath.Glob(filepath.Join(sysRoot, "bus/pci/devices/*"))
	devices := make([]PCIDevice, 0, len(paths))

	for _, path := range paths {
		vendor := read(filepath.Join(path, "vendor"))
		device := read(filepath.Join(path, "device"))

		if vendor == "" || device == "" {
			continue
		}

		devices = append(devices, PCIDevice{
			Address: filepath.Base(path),
			Vendor:  vendor,
			Device:  device,
			Class:   read(filepath.Join(path, "class")),
			Driver:  driverName(filepath.Join(path, "driver")),
		})
	}

	sort.Slice(devices, func(i, j int) bool {
		return devices[i].Address < devices[j].Address
	})

	return devices
}

func usbDevices(sysRoot string) []USBDevice {
	paths, _ := filepath.Glob(filepath.Join(sysRoot, "bus/usb/devices/*"))
	devices := make([]USBDevice, 0, len(paths))

	for _, path := range paths {
		vendor := read(filepath.Join(path, "idVendor"))
		product := read(filepath.Join(path, "idProduct"))

		if vendor == "" || product == "" {
			continue
		}

		devices = append(devices, USBDevice{
			Path:    filepath.Base(path),
			Vendor:  strings.ToLower(vendor),
			Product: strings.ToLower(product),
			Class:   read(filepath.Join(path, "bDeviceClass")),
			Driver:  driverName(filepath.Join(path, "driver")),
		})
	}

	sort.Slice(devices, func(i, j int) bool {
		return devices[i].Path < devices[j].Path
	})

	return devices
}

func driverName(path string) string {
	target, err := os.Readlink(path)
	if err != nil {
		return ""
	}
	return filepath.Base(target)
}
