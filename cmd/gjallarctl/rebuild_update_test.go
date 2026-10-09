package main

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@t"}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestPullCheckoutFastForwards(t *testing.T) {
	root := t.TempDir()
	upstream, checkout := filepath.Join(root, "up"), filepath.Join(root, "co")
	gitRun(t, root, "init", "-q", "-b", "main", upstream)
	gitRun(t, upstream, "commit", "-q", "--allow-empty", "-m", "one")
	gitRun(t, root, "clone", "-q", upstream, checkout)
	gitRun(t, upstream, "commit", "-q", "--allow-empty", "-m", "two")

	var stdout, stderr bytes.Buffer
	if code := pullCheckout(checkout, &stdout, &stderr); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "->") {
		t.Fatalf("head did not move: %q", stdout.String())
	}

	// A local commit makes the pull non-fast-forward: stop, rebuild nothing.
	gitRun(t, checkout, "commit", "-q", "--allow-empty", "-m", "local")
	gitRun(t, upstream, "commit", "-q", "--allow-empty", "-m", "three")
	stdout.Reset()
	stderr.Reset()
	if code := pullCheckout(checkout, &stdout, &stderr); code == 0 {
		t.Fatal("diverged checkout pulled")
	}
	if !strings.Contains(stderr.String(), "nothing rebuilt") {
		t.Fatalf("stderr %q", stderr.String())
	}
}

func TestRebuildHelpListsUpdate(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runRebuild([]string{"--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stdout.String(), "--update ") {
		t.Fatalf("help: %q", stdout.String())
	}
}
