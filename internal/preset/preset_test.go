package preset

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTypedValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preset.json")
	if err := os.WriteFile(path, []byte(`{"name":"testuser","enabled":true,"items":["one","two"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	document, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	name, _ := document.String("name")
	enabled, _ := document.Bool("enabled")
	items, _ := document.Strings("items")
	if name != "testuser" || !enabled || len(items) != 2 {
		t.Fatalf("unexpected values: %q %t %#v", name, enabled, items)
	}
}

func TestRejectsTrailingDocument(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preset.json")
	if err := os.WriteFile(path, []byte(`{} {}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("trailing JSON must fail")
	}
}
