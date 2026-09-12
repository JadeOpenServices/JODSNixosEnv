package deviceprobe

import (
	"path/filepath"
	"sort"
)

type ThunderboltDevice struct {
	Path       string `json:"path"`
	DeviceName string `json:"deviceName,omitempty"`
	VendorName string `json:"vendorName,omitempty"`
	Generation string `json:"generation,omitempty"`
	Authorized string `json:"authorized,omitempty"`
}

func thunderboltDevices(sysRoot string) []ThunderboltDevice {
	paths, _ := filepath.Glob(
		filepath.Join(sysRoot, "bus/thunderbolt/devices/*"),
	)

	devices := make([]ThunderboltDevice, 0, len(paths))

	for _, path := range paths {
		deviceName := read(filepath.Join(path, "device_name"))
		vendorName := read(filepath.Join(path, "vendor_name"))
		generation := read(filepath.Join(path, "generation"))
		authorized := read(filepath.Join(path, "authorized"))

		if deviceName == "" &&
			vendorName == "" &&
			generation == "" &&
			authorized == "" {
			continue
		}

		devices = append(devices, ThunderboltDevice{
			Path:       filepath.Base(path),
			DeviceName: deviceName,
			VendorName: vendorName,
			Generation: generation,
			Authorized: authorized,
		})
	}

	sort.Slice(devices, func(i, j int) bool {
		return devices[i].Path < devices[j].Path
	})

	return devices
}

type BlockDevice struct {
	Name      string `json:"name"`
	Removable bool   `json:"removable"`
	Vendor    string `json:"vendor,omitempty"`
	Model     string `json:"model,omitempty"`
	Driver    string `json:"driver,omitempty"`
}

func blockDevices(sysRoot string) []BlockDevice {
	paths, _ := filepath.Glob(filepath.Join(sysRoot, "class/block/*"))
	devices := make([]BlockDevice, 0, len(paths))

	for _, path := range paths {
		if read(filepath.Join(path, "partition")) != "" {
			continue
		}

		devices = append(devices, BlockDevice{
			Name:      filepath.Base(path),
			Removable: read(filepath.Join(path, "removable")) == "1",
			Vendor:    read(filepath.Join(path, "device/vendor")),
			Model:     read(filepath.Join(path, "device/model")),
			Driver:    driverName(filepath.Join(path, "device/driver")),
		})
	}

	sort.Slice(devices, func(i, j int) bool {
		return devices[i].Name < devices[j].Name
	})

	return devices
}
