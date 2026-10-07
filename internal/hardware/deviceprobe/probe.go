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
	Schema          int                     `json:"schema"`
	SysVendor       string                  `json:"sysVendor"`
	ProductName     string                  `json:"productName"`
	BoardName       string                  `json:"boardName"`
	PCI             []string                `json:"pci"`
	USB             []string                `json:"usb"`
	PCIDevices      []PCIDevice             `json:"pciDevices"`
	USBDevices      []USBDevice             `json:"usbDevices"`
	HIDDevices      []HIDDevice             `json:"hidDevices"`
	Input           []InputDevice           `json:"input"`
	IIO             []IIODevice             `json:"iio"`
	Thunderbolt     []ThunderboltDevice     `json:"thunderbolt"`
	ThunderboltHost bool                    `json:"thunderboltHost"`
	Block           []BlockDevice           `json:"block"`
	Audio           []AudioDevice           `json:"audio"`
	Fingerprint     FingerprintState        `json:"fingerprint"`
	PowerSupplies   []PowerSupply           `json:"powerSupplies"`
	Backlights      []BacklightDevice       `json:"backlights"`
	VideoDevices    []VideoDevice           `json:"videoDevices"`
	EmbeddedControl EmbeddedControllerState `json:"embeddedController"`
	Fans            []FanDevice             `json:"fans"`
	AMDPower        AMDPowerState           `json:"amdPower"`
}

// CollectLocal gathers hardware facts exposed directly by the kernel/sysfs.
// It deliberately performs no external command or D-Bus calls, so callers such
// as the installer can reuse the canonical hardware inventory independently of
// optional userspace tooling.
func CollectLocal(sysRoot string) Snapshot {
	s := Snapshot{
		Schema:          1,
		SysVendor:       read(filepath.Join(sysRoot, "class/dmi/id/sys_vendor")),
		ProductName:     read(filepath.Join(sysRoot, "class/dmi/id/product_name")),
		BoardName:       read(filepath.Join(sysRoot, "class/dmi/id/board_name")),
		PCI:             []string{},
		USB:             []string{},
		PCIDevices:      nonNilPCIDevices(pciDevices(sysRoot)),
		USBDevices:      nonNilUSBDevices(usbDevices(sysRoot)),
		HIDDevices:      nonNilHIDDevices(hidDevices(sysRoot)),
		Input:           nonNilInputDevices(inputDevices(sysRoot)),
		IIO:             nonNilIIODevices(iioDevices(sysRoot)),
		Thunderbolt:     nonNilThunderboltDevices(thunderboltDevices(sysRoot)),
		Block:           nonNilBlockDevices(blockDevices(sysRoot)),
		Audio:           nonNilAudioDevices(audioDevices(sysRoot)),
		Fingerprint:     FingerprintState{Devices: []FingerprintDevice{}},
		PowerSupplies:   nonNilPowerSupplies(powerSupplies(sysRoot)),
		Backlights:      nonNilBacklights(backlightDevices(sysRoot)),
		VideoDevices:    nonNilVideoDevices(videoDevices(sysRoot)),
		EmbeddedControl: embeddedControllerState(sysRoot),
		Fans:            nonNilFans(fanDevices(sysRoot)),
		AMDPower:        amdPowerState(sysRoot),
	}

	s.ThunderboltHost = len(s.Thunderbolt) != 0

	return s
}

func Collect(ctx context.Context, sysRoot string) (Snapshot, error) {
	s := CollectLocal(sysRoot)
	s.Fingerprint = fingerprintState(ctx)

	if !s.ThunderboltHost {
		if out, err := command(ctx, "boltctl", "domains"); err == nil {
			for _, line := range lines(out) {
				if strings.Contains(strings.ToLower(line), "domain") {
					s.ThunderboltHost = true
					break
				}
			}
		}
	}

	pci, err := command(ctx, "lspci", "-Dnn")
	if err != nil {
		return Snapshot{}, err
	}
	s.PCI = lines(pci)

	usb, err := command(ctx, "lsusb")
	if err != nil && hasUSBBus(sysRoot) {
		return Snapshot{}, err
	}
	// lsusb exits 1 on a machine without any USB bus (e2e-fw13 VM,
	// 2026-10-07); that is an empty list, not a failed probe.
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

func nonNilPowerSupplies(value []PowerSupply) []PowerSupply {
	if value == nil {
		return []PowerSupply{}
	}
	return value
}

func nonNilBacklights(value []BacklightDevice) []BacklightDevice {
	if value == nil {
		return []BacklightDevice{}
	}
	return value
}

func nonNilVideoDevices(value []VideoDevice) []VideoDevice {
	if value == nil {
		return []VideoDevice{}
	}
	return value
}

func nonNilFans(value []FanDevice) []FanDevice {
	if value == nil {
		return []FanDevice{}
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

func hasUSBBus(sysRoot string) bool {
	entries, err := os.ReadDir(filepath.Join(sysRoot, "bus/usb/devices"))
	return err != nil || len(entries) > 0
}
