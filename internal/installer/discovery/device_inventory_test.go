package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func writeInventoryFact(t *testing.T, path, value string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(value+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestDetectHardwareCarriesLocalDeviceInventory(t *testing.T) {
	root := t.TempDir()

	usb := filepath.Join(root, "bus", "usb", "devices", "1-4")
	writeInventoryFact(t, filepath.Join(usb, "idVendor"), "27c6")
	writeInventoryFact(t, filepath.Join(usb, "idProduct"), "609c")
	writeInventoryFact(t, filepath.Join(usb, "bDeviceClass"), "00")

	video := filepath.Join(root, "class", "video4linux", "video0")
	writeInventoryFact(t, filepath.Join(video, "name"), "Integrated Camera")

	got := DetectHardware(root)

	if got.DeviceInventory.Schema != 1 {
		t.Fatalf("inventory schema = %d, want 1", got.DeviceInventory.Schema)
	}

	if len(got.DeviceInventory.USBDevices) != 1 {
		t.Fatalf(
			"USB device count = %d, want 1",
			len(got.DeviceInventory.USBDevices),
		)
	}
	if got.DeviceInventory.USBDevices[0].Vendor != "27c6" ||
		got.DeviceInventory.USBDevices[0].Product != "609c" {
		t.Fatalf(
			"unexpected USB identity: %+v",
			got.DeviceInventory.USBDevices[0],
		)
	}

	if len(got.DeviceInventory.VideoDevices) != 1 ||
		got.DeviceInventory.VideoDevices[0].Name != "Integrated Camera" {
		t.Fatalf(
			"unexpected video inventory: %+v",
			got.DeviceInventory.VideoDevices,
		)
	}
}
