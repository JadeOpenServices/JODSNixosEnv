package profile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSelect(t *testing.T) {
	tests := []struct {
		name     string
		hardware Hardware
		override Override
		profile  string
		model    string
		context  int
	}{
		{"low memory", Hardware{RAMGB: 8}, Override{}, "low-memory", "qwen2.5-coder:7b", 8192},
		{"integrated", Hardware{RAMGB: 16, CPUCores: 4}, Override{}, "integrated", "qwen2.5-coder:14b", 16384},
		{"dedicated", Hardware{RAMGB: 32, CPUCores: 8, GPUType: "dedicated", VRAMMB: 12288}, Override{}, "dedicated", "qwen3-coder:30b", 32768},
		{"override", Hardware{RAMGB: 16, CPUCores: 4}, Override{Enabled: true, Model: "mistral"}, "user-override", "mistral", 16384},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Select(test.hardware, test.override)
			if got.Profile != test.profile || got.Model != test.model || got.ContextTokens != test.context {
				t.Fatalf("Select() = %#v", got)
			}
		})
	}
}

func TestLoadOverrideRequiresModel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "user.config.json")
	if err := os.WriteFile(path, []byte(`{"overrideAiSelection":true,"overrideModelWith":""}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadOverride(path); err == nil {
		t.Fatal("empty manual override accepted")
	}
}

func TestLoadOverrideRejectsControlCharacters(t *testing.T) {
	path := filepath.Join(t.TempDir(), "user.config.json")
	if err := os.WriteFile(path, []byte(`{"overrideAiSelection":true,"overrideModelWith":"bad\nmodel"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadOverride(path); err == nil {
		t.Fatal("control characters must be rejected")
	}
}
