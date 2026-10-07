package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadIgnoresLegacyBackgroundNormal(t *testing.T) {
	preset, err := os.ReadFile("../../../scripts/installation/user_PresetJSON/default.user.config.json")
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(preset, &fields); err != nil {
		t.Fatal(err)
	}
	fields["backgroundNormal"] = "/home/user/Pictures/old.jpg"
	contents, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "user.config.json")
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err != nil {
		t.Fatalf("Load() rejected the old backgroundNormal key: %v", err)
	}
}
