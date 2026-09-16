package deviceprobe

import (
	"encoding/json"
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
