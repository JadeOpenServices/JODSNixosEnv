package deploy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTarget(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "flake.nix"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	got, err := Target(root, "gjallarOS")
	if err != nil || got != "path:"+root+"#gjallarOS" {
		t.Fatalf("got %q, %v", got, err)
	}
	for _, hostname := range []string{"", "bad name", "../bad"} {
		if _, err := Target(root, hostname); err == nil {
			t.Fatalf("accepted %q", hostname)
		}
	}
}

func TestEvalSwapSize(t *testing.T) {
	small := []byte("MemTotal:        3923456 kB\nSwapTotal:             0 kB\n")
	if got := evalSwapSize(false, small); got != 3923456<<10 {
		t.Fatalf("4 GiB machine: %d", got)
	}
	if got := evalSwapSize(true, small); got != 0 {
		t.Fatalf("swap already active: %d", got)
	}
	large := []byte("MemTotal:       65000000 kB\n")
	if got := evalSwapSize(false, large); got != 8<<30 {
		t.Fatalf("64 GiB machine not capped: %d", got)
	}
	if got := evalSwapSize(false, []byte("garbage")); got != 0 {
		t.Fatalf("unreadable meminfo: %d", got)
	}
}
