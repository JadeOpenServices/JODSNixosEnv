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
	root := testRepository(t, `{"hostname":"gjallarOS"}`)
	report := Check(context.Background(), root)
	if report.Failed() {
		t.Fatalf("valid minimal config should not fail: %#v", report.Findings)
	}
}

func TestCheckRejectsWrongPresetFieldType(t *testing.T) {
	root := testRepository(t, `{"hostname":false}`)
	report := Check(context.Background(), root)
	if !report.Failed() {
		t.Fatal("wrong preset field type must fail")
	}
}

func testRepository(t *testing.T, config string) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"flake.nix": `
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs";
  };
  outputs = { self, nixpkgs }: { test = nixpkgs; };
`,
		"user.config.json":                config,
		"scripts/installation/install.sh": "#!/usr/bin/env bash\n",
		"scripts/installation/user_PresetJSON/default.user.config.json": `{"hostname":"gjallarOS"}`,
		"cmd/gjallar-installer/main.go":                                 "package main\n",
		"internal/installer/app/app.go":                                 "package app\n",
		"generated/state.nix":                                           "{}\n",
		"generated/hardware.nix":                                        "{}\n",
		"generated/install-state.nix":                                   "{}\n",
		"system/default.nix":                                            "{}\n",
		"apps/ai/nixos.nix":                                             "{}\n",
		"user/default.nix":                                              "{}\n",
		"system/tools/commands/default.nix":                             "# check-installer\n",
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

func TestPreflightWarnsAboutUnusedFlakeInput(t *testing.T) {
	root := testRepository(t, `{"hostname":"gjallarOS"}`)
	flake := `
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs";
    dead-input.url = "github:example/dead";
  };
  outputs = { self, nixpkgs, ... }@inputs: { test = inputs.nixpkgs; };
`
	if err := os.WriteFile(filepath.Join(root, "flake.nix"), []byte(flake), 0o600); err != nil {
		t.Fatal(err)
	}

	report := Preflight(root)
	if !hasFinding(report, Warn, "unused flake input: dead-input") {
		t.Fatalf("missing dead-input warning: %#v", report.Findings)
	}
}

func TestPreflightWarnsAboutUnwiredModule(t *testing.T) {
	root := testRepository(t, `{"hostname":"gjallarOS"}`)
	path := filepath.Join(root, "system", "compat", "orphan.nix")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	report := Preflight(root)
	if !hasFinding(report, Warn, "likely unwired Nix module: system/compat/orphan.nix") {
		t.Fatalf("missing orphan warning: %#v", report.Findings)
	}
}

func TestPreflightAllowsExplicitDormantModule(t *testing.T) {
	root := testRepository(t, `{"hostname":"gjallarOS"}`)
	path := filepath.Join(root, "user", "apps", "dormant", "default.nix")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		path,
		[]byte("# gjallar: dormant-module\n{}\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	report := Preflight(root)
	message := "likely unwired Nix module: user/apps/dormant/default.nix"
	if hasFinding(report, Warn, message) {
		t.Fatalf("dormant module reported as unwired: %#v", report.Findings)
	}
}

func hasFinding(report Report, level Level, message string) bool {
	for _, finding := range report.Findings {
		if finding.Level == level && finding.Message == message {
			return true
		}
	}
	return false
}
