package deviceprobe

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Snapshot struct {
	Schema      int                 `json:"schema"`
	SysVendor   string              `json:"sysVendor"`
	ProductName string              `json:"productName"`
	BoardName   string              `json:"boardName"`
	PCI         []string            `json:"pci"`
	USB         []string            `json:"usb"`
	PCIDevices  []PCIDevice         `json:"pciDevices"`
	USBDevices  []USBDevice         `json:"usbDevices"`
	HIDDevices  []HIDDevice         `json:"hidDevices"`
	Input       []InputDevice       `json:"input"`
	IIO         []IIODevice         `json:"iio"`
	Thunderbolt []ThunderboltDevice `json:"thunderbolt"`
	Block       []BlockDevice       `json:"block"`
	Audio       []AudioDevice       `json:"audio"`
	Fingerprint FingerprintState    `json:"fingerprint"`
}

func Collect(ctx context.Context, sysRoot string) (Snapshot, error) {
	s := Snapshot{
		Schema:      1,
		SysVendor:   read(filepath.Join(sysRoot, "class/dmi/id/sys_vendor")),
		ProductName: read(filepath.Join(sysRoot, "class/dmi/id/product_name")),
		BoardName:   read(filepath.Join(sysRoot, "class/dmi/id/board_name")),
		PCI:         []string{},
		USB:         []string{},
		PCIDevices:  nonNilPCIDevices(pciDevices(sysRoot)),
		USBDevices:  nonNilUSBDevices(usbDevices(sysRoot)),
		HIDDevices:  nonNilHIDDevices(hidDevices(sysRoot)),
		Input:       nonNilInputDevices(inputDevices(sysRoot)),
		IIO:         nonNilIIODevices(iioDevices(sysRoot)),
		Thunderbolt: nonNilThunderboltDevices(thunderboltDevices(sysRoot)),
		Block:       nonNilBlockDevices(blockDevices(sysRoot)),
		Audio:       nonNilAudioDevices(audioDevices(sysRoot)),
		Fingerprint: fingerprintState(ctx),
	}

	pci, err := command(ctx, "lspci", "-Dnn")
	if err != nil {
		return Snapshot{}, err
	}
	s.PCI = lines(pci)

	usb, err := command(ctx, "lsusb")
	if err != nil {
		return Snapshot{}, err
	}
	s.USB = lines(usb)

	return s, nil
}

func nonNilPCIDevices(value []PCIDevice) []PCIDevice {
	if value == nil {
		return []PCIDevice{}
	}
	return value
}

func nonNilUSBDevices(value []USBDevice) []USBDevice {
	if value == nil {
		return []USBDevice{}
	}
	return value
}

func nonNilInputDevices(value []InputDevice) []InputDevice {
	if value == nil {
		return []InputDevice{}
	}
	return value
}

func nonNilIIODevices(value []IIODevice) []IIODevice {
	if value == nil {
		return []IIODevice{}
	}
	return value
}

func nonNilThunderboltDevices(value []ThunderboltDevice) []ThunderboltDevice {
	if value == nil {
		return []ThunderboltDevice{}
	}
	return value
}

func nonNilBlockDevices(value []BlockDevice) []BlockDevice {
	if value == nil {
		return []BlockDevice{}
	}
	return value
}

func nonNilAudioDevices(value []AudioDevice) []AudioDevice {
	if value == nil {
		return []AudioDevice{}
	}
	return value
}

func read(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func command(ctx context.Context, name string, args ...string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("%s unavailable", name)
	}

	out, err := exec.CommandContext(ctx, path, args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s failed: %w", name, err)
	}

	return string(out), nil
}

func lines(value string) []string {
	var out []string
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}
