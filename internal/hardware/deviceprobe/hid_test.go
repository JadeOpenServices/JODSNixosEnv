package deviceprobe

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHIDDevicesCollectIdentityAndHidraw(t *testing.T) {
	root := t.TempDir()

	hid := filepath.Join(
		root,
		"bus/hid/devices/0003:03F0:0A56.0001",
	)
	if err := os.MkdirAll(hid, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(hid, "uevent"),
		[]byte(
			"HID_ID=0003:000003F0:00000A56\n"+
				"HID_NAME=HP Inc. HP ZBook Create x2 Quick Keys\n",
		),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	driver := filepath.Join(root, "drivers/hid-generic")
	if err := os.MkdirAll(driver, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(driver, filepath.Join(hid, "driver")); err != nil {
		t.Fatal(err)
	}

	hidraw := filepath.Join(root, "class/hidraw/hidraw0")
	if err := os.MkdirAll(hidraw, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(hid, filepath.Join(hidraw, "device")); err != nil {
		t.Fatal(err)
	}

	got := hidDevices(root)
	if len(got) != 1 {
		t.Fatalf("got %d HID devices, want 1", len(got))
	}

	device := got[0]

	if device.Bus != "0003" ||
		device.Vendor != "03f0" ||
		device.Product != "0a56" {
		t.Fatalf("unexpected HID identity: %+v", device)
	}

	if device.Name != "HP Inc. HP ZBook Create x2 Quick Keys" {
		t.Fatalf("unexpected HID name %q", device.Name)
	}

	if device.Driver != "hid-generic" {
		t.Fatalf("unexpected driver %q", device.Driver)
	}

	if len(device.Hidraw) != 1 || device.Hidraw[0] != "hidraw0" {
		t.Fatalf("unexpected hidraw nodes: %#v", device.Hidraw)
	}
}

func TestHIDDevicesWithoutHidrawUseEmptyArray(t *testing.T) {
	root := t.TempDir()

	hid := filepath.Join(
		root,
		"bus/hid/devices/0018:056A:016C.0007",
	)
	if err := os.MkdirAll(hid, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(hid, "uevent"),
		[]byte(
			"HID_ID=0018:0000056A:0000016C\n"+
				"HID_NAME=WCOM002E:00 056A:016C\n",
		),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	got := hidDevices(root)
	if len(got) != 1 {
		t.Fatalf("got %d HID devices, want 1", len(got))
	}

	if got[0].Hidraw == nil {
		t.Fatal("hidraw must be an empty array, not nil")
	}

	if len(got[0].Hidraw) != 0 {
		t.Fatalf("unexpected hidraw nodes: %#v", got[0].Hidraw)
	}
}

func TestHIDDeviceJSONUsesEmptyHidrawArray(t *testing.T) {
	device := HIDDevice{
		Path:    "0018:056A:016C.0007",
		Bus:     "0018",
		Vendor:  "056a",
		Product: "016c",
		Name:    "WCOM002E:00 056A:016C",
		Hidraw:  []string{},
	}

	data, err := json.Marshal(device)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(data), `"hidraw":[]`) {
		t.Fatalf("hidraw did not marshal as empty array: %s", data)
	}
}
