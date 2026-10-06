package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fakeODDCctl(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "oddcctl")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho \"$@\"\nexit 3\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	oldPath, oldRoot := oddcctlPath, oddcRoot
	oddcctlPath, oddcRoot = script, "/catalog"
	t.Cleanup(func() { oddcctlPath, oddcRoot = oldPath, oldRoot })
}

func TestODDCPassthroughAddsSystemRoot(t *testing.T) {
	fakeODDCctl(t)
	var stdout, stderr bytes.Buffer
	code := runODDC([]string{"list", "--kind", "DeviceModel"}, &stdout, &stderr)
	if code != 3 {
		t.Fatalf("exit code = %d, want oddcctl's 3; stderr %q", code, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "list --kind DeviceModel --root /catalog" {
		t.Fatalf("argv = %q", got)
	}
}

func TestODDCPassthroughKeepsExplicitRoot(t *testing.T) {
	fakeODDCctl(t)
	var stdout, stderr bytes.Buffer
	runODDC([]string{"resolve", "--root", "./mine", "--device", "model/hp/zbook-x2-g4"}, &stdout, &stderr)
	if got := strings.TrimSpace(stdout.String()); got != "resolve --root ./mine --device model/hp/zbook-x2-g4" {
		t.Fatalf("argv = %q", got)
	}
}

func TestODDCRejectsUnknownCommand(t *testing.T) {
	fakeODDCctl(t)
	var stdout, stderr bytes.Buffer
	if code := runODDC([]string{"delete"}, &stdout, &stderr); code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("oddcctl ran: %q", stdout.String())
	}
}

func TestODDCResolveDefaultsToDeployedModel(t *testing.T) {
	fakeODDCctl(t)
	root := t.TempDir()
	oddcRoot = root
	if err := os.WriteFile(filepath.Join(root, "resolved.json"), []byte(`{"model":{"id":"model/framework/laptop-13-amd-ryzen-7040"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	overlay := filepath.Join(root, "host-overlay.json")
	if err := os.WriteFile(overlay, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	runODDC([]string{"resolve"}, &stdout, &stderr)
	want := "resolve --root " + root + " --device model/framework/laptop-13-amd-ryzen-7040 --host " + overlay
	if got := strings.TrimSpace(stdout.String()); got != want {
		t.Fatalf("argv = %q, want %q", got, want)
	}
}
