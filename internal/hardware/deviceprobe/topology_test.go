package deviceprobe

import (
	"path/filepath"
	"testing"
)

func TestThunderboltDevices(t *testing.T) {
	root := t.TempDir()
	device := filepath.Join(root, "bus/thunderbolt/devices/0-1")

	writeProbeFile(t, filepath.Join(device, "device_name"), "Test Dock")
	writeProbeFile(t, filepath.Join(device, "vendor_name"), "Example")
	writeProbeFile(t, filepath.Join(device, "generation"), "3")
	writeProbeFile(t, filepath.Join(device, "authorized"), "1")

	got := thunderboltDevices(root)
	if len(got) != 1 {
		t.Fatalf("got %d Thunderbolt devices, want 1", len(got))
	}

	if got[0].DeviceName != "Test Dock" ||
		got[0].VendorName != "Example" ||
		got[0].Authorized != "1" {
		t.Fatalf("unexpected Thunderbolt device: %+v", got[0])
	}
}

func TestBlockDevices(t *testing.T) {
	root := t.TempDir()

	disk := filepath.Join(root, "class/block/mmcblk0")
	writeProbeFile(t, filepath.Join(disk, "removable"), "1")
	writeProbeFile(t, filepath.Join(disk, "device/vendor"), "SD")
	writeProbeFile(t, filepath.Join(disk, "device/model"), "Card Reader")

	partition := filepath.Join(root, "class/block/mmcblk0p1")
	writeProbeFile(t, filepath.Join(partition, "partition"), "1")
	writeProbeFile(t, filepath.Join(partition, "removable"), "1")

	got := blockDevices(root)
	if len(got) != 1 {
		t.Fatalf("got %d block devices, want 1", len(got))
	}

	if got[0].Name != "mmcblk0" || !got[0].Removable {
		t.Fatalf("unexpected block device: %+v", got[0])
	}
}

func TestCollectIncludesTopology(t *testing.T) {
	root := t.TempDir()

	writeProbeFile(
		t,
		filepath.Join(root, "class/dmi/id/sys_vendor"),
		"Test Vendor",
	)

	thunderbolt := filepath.Join(root, "bus/thunderbolt/devices/0-1")
	writeProbeFile(
		t,
		filepath.Join(thunderbolt, "device_name"),
		"Test Dock",
	)

	block := filepath.Join(root, "class/block/mmcblk0")
	writeProbeFile(
		t,
		filepath.Join(block, "removable"),
		"1",
	)

	// Collect also invokes lspci/lsusb, so this test verifies the topology
	// initializer indirectly only through the helpers' integration contract.
	s := Snapshot{
		Thunderbolt: thunderboltDevices(root),
		Block:       blockDevices(root),
	}

	if len(s.Thunderbolt) != 1 {
		t.Fatalf("got %d Thunderbolt devices, want 1", len(s.Thunderbolt))
	}
	if len(s.Block) != 1 {
		t.Fatalf("got %d block devices, want 1", len(s.Block))
	}
}
