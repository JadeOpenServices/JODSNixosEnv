package oddccli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeCommand stands in for a command: it appends its arguments to log.
func fakeCommand(t *testing.T, log, name string) string {
	t.Helper()

	script := filepath.Join(t.TempDir(), name)
	body := "#!/bin/sh\necho " + name + " \"$@\" >> " + log + "\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return script
}

func runUpdateWith(t *testing.T, args ...string) (int, []string) {
	t.Helper()

	log := filepath.Join(t.TempDir(), "log")
	oldNix, oldSelf := Nix, Gjallarctl
	Nix, Gjallarctl = fakeCommand(t, log, "nix"), fakeCommand(t, log, "gjallarctl")
	t.Cleanup(func() { Nix, Gjallarctl = oldNix, oldSelf })

	var stdout, stderr bytes.Buffer
	code := Run(append([]string{"update"}, args...), &stdout, &stderr)

	data, _ := os.ReadFile(log)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(data) == 0 {
		lines = nil
	}
	return code, lines
}

func TestUpdateMovesODDCInput(t *testing.T) {
	code, ran := runUpdateWith(t, "--repo", "/repo")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if want := []string{"nix flake update oddc --flake /repo"}; strings.Join(ran, "|") != strings.Join(want, "|") {
		t.Fatalf("ran %q, want %q", ran, want)
	}
}

func TestUpdateRebuilds(t *testing.T) {
	code, ran := runUpdateWith(t, "--repo", "/repo", "--rebuild")
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	want := []string{
		"nix flake update oddc --flake /repo",
		"gjallarctl rebuild --repo /repo",
	}
	if strings.Join(ran, "|") != strings.Join(want, "|") {
		t.Fatalf("ran %q, want %q", ran, want)
	}
}

func TestUpdateRejectsArguments(t *testing.T) {
	code, ran := runUpdateWith(t, "main")
	if code != 2 || ran != nil {
		t.Fatalf("exit code %d, ran %q; want 2 and nothing run", code, ran)
	}
}
