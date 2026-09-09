package localgit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContainedRelativeRejectsEscape(t *testing.T) {
	root := t.TempDir()
	if _, err := containedRelative(root, filepath.Join(root, "..", "outside")); err == nil {
		t.Fatal("expected outside path rejection")
	}
	got, err := containedRelative(root, filepath.Join(root, "profiles", "laptop", "hardware-configuration.nix"))
	if err != nil || got != filepath.Join("profiles", "laptop", "hardware-configuration.nix") {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestUpdateExcludeIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "info", "exclude")
	if err := updateExclude(path); err != nil {
		t.Fatal(err)
	}
	if err := updateExclude(path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := map[string]int{}
	for _, line := range strings.Split(string(data), "\n") {
		if line == "" {
			continue
		}
		lines[line]++
	}

	for _, pattern := range excludePatterns() {
		if lines[pattern] != 1 {
			t.Fatalf("pattern line count != 1: %s (%d)", pattern, lines[pattern])
		}
	}
}
