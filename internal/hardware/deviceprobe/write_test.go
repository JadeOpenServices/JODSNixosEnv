package deviceprobe

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteSnapshotCanonicalJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "device.json")

	want := Snapshot{
		SysVendor:   "HP",
		ProductName: "HP ZBook x2 G4",
		BoardName:   "824C",
		PCI:         []string{"0000:01:00.0 NVIDIA [10de:13b4]"},
		USB:         []string{"Bus 001 Device 002: ID 1234:5678"},
	}

	if err := WriteSnapshot(path, want); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 || data[len(data)-1] != '\n' {
		t.Fatal("snapshot does not end in one newline")
	}

	var got Snapshot
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}

	if got.ProductName != want.ProductName || got.BoardName != want.BoardName {
		t.Fatalf("snapshot mismatch: %+v", got)
	}
}

func TestReadSnapshot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "device-probe.json")

	want := Snapshot{
		Schema:      1,
		SysVendor:   "HP",
		ProductName: "HP ZBook x2 G4",
		BoardName:   "824C",
	}

	if err := WriteSnapshot(path, want); err != nil {
		t.Fatal(err)
	}

	got, err := ReadSnapshot(path)
	if err != nil {
		t.Fatal(err)
	}

	if got.SysVendor != want.SysVendor ||
		got.ProductName != want.ProductName ||
		got.BoardName != want.BoardName {
		t.Fatalf("snapshot mismatch: %+v", got)
	}
}
