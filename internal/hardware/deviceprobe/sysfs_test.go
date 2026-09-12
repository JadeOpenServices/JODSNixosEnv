package deviceprobe

import (
	"os"
	"path/filepath"
	"testing"
)

func writeProbeFile(t *testing.T, path, value string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(value+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestPCIDevices(t *testing.T) {
	root := t.TempDir()
	device := filepath.Join(root, "bus/pci/devices/0000:01:00.0")

	writeProbeFile(t, filepath.Join(device, "vendor"), "0x10de")
	writeProbeFile(t, filepath.Join(device, "device"), "0x13b4")
	writeProbeFile(t, filepath.Join(device, "class"), "0x030200")

	if err := os.Symlink(
		"../../../../bus/pci/drivers/nvidia",
		filepath.Join(device, "driver"),
	); err != nil {
		t.Fatal(err)
	}

	got := pciDevices(root)
	if len(got) != 1 {
		t.Fatalf("got %d PCI devices, want 1", len(got))
	}

	if got[0].Address != "0000:01:00.0" ||
		got[0].Vendor != "0x10de" ||
		got[0].Device != "0x13b4" ||
		got[0].Driver != "nvidia" {
		t.Fatalf("unexpected PCI device: %+v", got[0])
	}
}

func TestUSBDevices(t *testing.T) {
	root := t.TempDir()
	device := filepath.Join(root, "bus/usb/devices/1-2")

	writeProbeFile(t, filepath.Join(device, "idVendor"), "056a")
	writeProbeFile(t, filepath.Join(device, "idProduct"), "1234")
	writeProbeFile(t, filepath.Join(device, "bDeviceClass"), "00")

	if err := os.Symlink(
		"../../../../../bus/usb/drivers/usbhid",
		filepath.Join(device, "driver"),
	); err != nil {
		t.Fatal(err)
	}

	got := usbDevices(root)
	if len(got) != 1 {
		t.Fatalf("got %d USB devices, want 1", len(got))
	}

	if got[0].Path != "1-2" ||
		got[0].Vendor != "056a" ||
		got[0].Product != "1234" ||
		got[0].Driver != "usbhid" {
		t.Fatalf("unexpected USB device: %+v", got[0])
	}
}
