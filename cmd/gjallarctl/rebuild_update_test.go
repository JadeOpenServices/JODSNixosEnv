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

func TestODDCOverrideRevision(t *testing.T) {
	const sha = "9958ddcb93c26721a60f033e21cbb27c8c89325f"
	cases := []struct {
		args    []string
		want    string
		wantErr bool
	}{
		{nil, "", false},
		{[]string{"--show-trace"}, "", false},
		{[]string{"--override-input", "nixpkgs", "github:NixOS/nixpkgs/" + sha}, "", false},
		{[]string{"--override-input", "oddc", "github:JadeOpenServices/oddc/" + sha, "--no-write-lock-file"}, sha, false},
		{[]string{"--override-input", "oddc", "git+https://github.com/JadeOpenServices/oddc?ref=main&rev=" + sha}, sha, false},
		{[]string{"--override-input", "oddc", "github:JadeOpenServices/oddc/main"}, "", true},
		{[]string{"--override-input", "oddc", "path:/home/user/oddc"}, "", true},
	}
	for _, c := range cases {
		got, err := oddcOverrideRevision(c.args)
		if (err != nil) != c.wantErr || got != c.want {
			t.Errorf("oddcOverrideRevision(%q) = %q, %v; want %q, error %v", c.args, got, err, c.want, c.wantErr)
		}
	}
}
