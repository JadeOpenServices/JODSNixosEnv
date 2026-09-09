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
