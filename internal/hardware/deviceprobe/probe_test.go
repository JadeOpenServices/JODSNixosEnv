package deviceprobe

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLinesDropsEmptyRows(t *testing.T) {
	got := lines("one\n\n two \n")

	if len(got) != 2 || got[0] != "one" || got[1] != "two" {
		t.Fatalf("unexpected lines: %#v", got)
	}
}

func TestCollectSnapshotArraysMarshalAsArrays(t *testing.T) {
	s := Snapshot{
		Schema:        1,
		PCI:           []string{},
		USB:           []string{},
		PCIDevices:    []PCIDevice{},
		USBDevices:    []USBDevice{},
		HIDDevices:    []HIDDevice{},
		Input:         []InputDevice{},
		IIO:           []IIODevice{},
		Thunderbolt:   []ThunderboltDevice{},
		Block:         []BlockDevice{},
		Audio:         []AudioDevice{},
		PowerSupplies: []PowerSupply{},
		Backlights:    []BacklightDevice{},
		VideoDevices:  []VideoDevice{},
		Fans:          []FanDevice{},
	}

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}

	text := string(data)
	for _, field := range []string{
		`"pci":[]`,
		`"usb":[]`,
		`"pciDevices":[]`,
		`"usbDevices":[]`,
		`"hidDevices":[]`,
		`"input":[]`,
		`"iio":[]`,
		`"thunderbolt":[]`,
		`"block":[]`,
		`"audio":[]`,
		`"powerSupplies":[]`,
		`"backlights":[]`,
		`"videoDevices":[]`,
		`"fans":[]`,
	} {
		if !strings.Contains(text, field) {
			t.Fatalf("missing canonical empty array %s in %s", field, text)
		}
	}
}

func TestHasUSBBus(t *testing.T) {
	root := t.TempDir()
	if !hasUSBBus(root) {
		t.Fatal("unreadable bus/usb/devices must count as a bus, so lsusb errors still fail")
	}
	if err := os.MkdirAll(filepath.Join(root, "bus/usb/devices"), 0o755); err != nil {
		t.Fatal(err)
	}
	if hasUSBBus(root) {
		t.Fatal("empty bus/usb/devices reported a USB bus")
	}
	if err := os.WriteFile(filepath.Join(root, "bus/usb/devices/usb1"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if !hasUSBBus(root) {
		t.Fatal("bus/usb/devices/usb1 not reported as a USB bus")
	}
}
