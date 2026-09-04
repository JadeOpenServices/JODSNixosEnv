package installercheck

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveRepositoryRequiresMarkers(t *testing.T) {
	root := t.TempDir()
	if _, err := ResolveRepository(root); err == nil {
		t.Fatal("expected missing markers to be rejected")
	}
	for _, relative := range []string{"flake.nix", "scripts/installation/install.sh"} {
		path := filepath.Join(root, relative)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("test"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	got, err := ResolveRepository(root)
	if err != nil || got != root {
		t.Fatalf("ResolveRepository() = %q, %v", got, err)
	}
}

func TestCheckRejectsInvalidUserConfig(t *testing.T) {
	root := testRepository(t, "{")
	report := Check(context.Background(), root)
	if !report.Failed() {
		t.Fatal("invalid user config must fail")
	}
}

func TestCheckAcceptsInstallerDefaults(t *testing.T) {
	root := testRepository(t, `{"system":"x86_64-linux"}`)
	report := Check(context.Background(), root)
	if report.Failed() {
		t.Fatalf("valid minimal config should not fail: %#v", report.Findings)
	}
}

func TestCheckRejectsWrongPresetFieldType(t *testing.T) {
	root := testRepository(t, `{"system":false}`)
	report := Check(context.Background(), root)
	if !report.Failed() {
		t.Fatal("wrong preset field type must fail")
	}
}

func testRepository(t *testing.T, config string) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"flake.nix":                       "{}",
		"user.config.json":                config,
		"scripts/installation/install.sh": "#!/usr/bin/env bash\n",
		"scripts/installation/user_PresetJSON/default.user.config.json":   `{"system":"x86_64-linux","profile":"laptop"}`,
		"scripts/installation/functions/configuration/render_settings.sh": "render_settings() {}\n",
		"scripts/installation/functions/hardware/detect_graphics.sh":      "detect_graphics() { :; }\n",
		"system/apps/ollama.nix":           "{}\n",
		"system/tools/scripts/default.nix": "# check-installer\n",
		"system/tools/scripts/help.sh":     "# check-installer\n",
	}
	for relative, contents := range files {
		path := filepath.Join(root, relative)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
