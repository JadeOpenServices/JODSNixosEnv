package localgit

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestTrackedPathRejectsGeneratedUntrackedHardwareConfig(t *testing.T) {
	root := t.TempDir()

	if out, err := exec.Command("git", "-C", root, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}

	relative := filepath.Join(
		"profiles",
		"laptop",
		"hardware-configuration.nix",
	)
	path := filepath.Join(root, relative)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if trackedPath(context.Background(), root, relative) {
		t.Fatal("untracked generated hardware configuration reported as tracked")
	}

	if out, err := exec.Command(
		"git", "-C", root, "add", "--", relative,
	).CombinedOutput(); err != nil {
		t.Fatalf("git add: %v: %s", err, out)
	}

	if !trackedPath(context.Background(), root, relative) {
		t.Fatal("indexed hardware configuration was not reported as tracked")
	}
}
