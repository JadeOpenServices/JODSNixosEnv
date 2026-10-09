package app

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func freshCheckoutFixture(t *testing.T) (repo, target string) {
	t.Helper()
	repo, target = t.TempDir(), t.TempDir()
	files := map[string]os.FileMode{
		"flake.nix":                0o644,
		".git/HEAD":                0o644,
		"generated/hardware.nix":   0o600,
		"user.config.json":         0o600,
		"scripts/run.sh":           0o755,
		".vm/e2e/disk.qcow2":       0o644,
		".claude/settings.json":    0o644,
		"pkgs/monique/default.nix": 0o644,
	}
	for rel, mode := range files {
		path := filepath.Join(repo, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(rel), mode); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink("/nix/store/x-system", filepath.Join(repo, "result")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(target, "etc"), 0o755); err != nil {
		t.Fatal(err)
	}
	passwd := "root:x:0:0:System administrator:/root:/bin/sh\n" +
		"tester:x:1000:100::/home/tester:/run/current-system/sw/bin/zsh\n"
	if err := os.WriteFile(filepath.Join(target, "etc", "passwd"), []byte(passwd), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(target, "home", "tester"), 0o700); err != nil {
		t.Fatal(err)
	}
	return repo, target
}

func TestCopyFreshCheckoutPlacesCheckoutAtDotfilesPath(t *testing.T) {
	repo, target := freshCheckoutFixture(t)
	calls := withPrivilegedCommand(t, runUnprivileged)

	if err := copyFreshCheckout(context.Background(), repo, target, "/home/tester/Documents/gjallarOS", "tester", io.Discard); err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(target, "home", "tester", "Documents", "gjallarOS")
	for rel, mode := range map[string]os.FileMode{
		"flake.nix":              0o644,
		".git/HEAD":              0o644,
		"generated/hardware.nix": 0o600,
		"user.config.json":       0o600,
		"scripts/run.sh":         0o755,
	} {
		info, err := os.Stat(filepath.Join(dest, rel))
		if err != nil {
			t.Errorf("%s not copied: %v", rel, err)
			continue
		}
		if info.Mode().Perm() != mode {
			t.Errorf("%s mode %o, want %o", rel, info.Mode().Perm(), mode)
		}
	}
	for _, drop := range []string{".vm", ".claude", "result"} {
		if _, err := os.Lstat(filepath.Join(dest, drop)); !os.IsNotExist(err) {
			t.Errorf("%s was copied: %v", drop, err)
		}
	}

	var chowns []string
	for _, c := range *calls {
		if c[0] == "chown" {
			chowns = append(chowns, strings.Join(c, " "))
		}
	}
	home := filepath.Join(target, "home", "tester")
	want := []string{
		"chown -h 1000:100 -- " + home + " " + filepath.Join(home, "Documents"),
		"chown -R -P -h 1000:100 -- " + dest,
	}
	if !slices.Equal(chowns, want) {
		t.Fatalf("chown calls:\n%s\nwant:\n%s", strings.Join(chowns, "\n"), strings.Join(want, "\n"))
	}
}

func TestCopyFreshCheckoutOutsideHomeKeepsParentsRoot(t *testing.T) {
	repo, target := freshCheckoutFixture(t)
	calls := withPrivilegedCommand(t, runUnprivileged)

	if err := copyFreshCheckout(context.Background(), repo, target, "/opt/gjallarOS", "tester", io.Discard); err != nil {
		t.Fatal(err)
	}
	for _, c := range *calls {
		if c[0] == "chown" && !slices.Contains(c, "-R") {
			t.Fatalf("parent outside the home handed to the user: %v", c)
		}
	}
}

func TestCopyFreshCheckoutNeverMergesIntoExistingDirectory(t *testing.T) {
	repo, target := freshCheckoutFixture(t)
	existing := filepath.Join(target, "home", "tester", "Documents", "gjallarOS")
	if err := os.MkdirAll(existing, 0o755); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(existing, "keep")
	if err := os.WriteFile(keep, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	calls := withPrivilegedCommand(t, runUnprivileged)

	if err := copyFreshCheckout(context.Background(), repo, target, "/home/tester/Documents/gjallarOS", "tester", io.Discard); err == nil {
		t.Fatal("copied into an existing directory")
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatalf("existing directory was removed: %v", err)
	}
	for _, c := range *calls {
		if c[0] == "cp" || c[0] == "rm" {
			t.Fatalf("ran %v after the destination check failed", c)
		}
	}
}

func TestCopyFreshCheckoutRemovesPartialCopy(t *testing.T) {
	repo, target := freshCheckoutFixture(t)
	calls := withPrivilegedCommand(t, func(args []string) ([]byte, error) {
		if args[0] == "cp" {
			return nil, os.ErrPermission
		}
		return runUnprivileged(args)
	})

	if err := copyFreshCheckout(context.Background(), repo, target, "/home/tester/Documents/gjallarOS", "tester", io.Discard); err == nil {
		t.Fatal("failed copy was ignored")
	}
	last := (*calls)[len(*calls)-1]
	dest := filepath.Join(target, "home", "tester", "Documents", "gjallarOS")
	if strings.Join(last, " ") != "rm -rf -- "+dest {
		t.Fatalf("partial copy left behind; last command %v", last)
	}
}

func TestCopyFreshCheckoutRejectsBadInput(t *testing.T) {
	repo, target := freshCheckoutFixture(t)
	calls := withPrivilegedCommand(t, nil)
	for _, tc := range []struct{ dir, user string }{
		{"Documents/gjallarOS", "tester"},
		{"/", "tester"},
		{"/home/ghost/Documents/gjallarOS", "ghost"},
		{"/root/gjallarOS", "root"},
	} {
		if err := copyFreshCheckout(context.Background(), repo, target, tc.dir, tc.user, io.Discard); err == nil {
			t.Errorf("accepted dir %q user %q", tc.dir, tc.user)
		}
	}
	if len(*calls) != 0 {
		t.Fatalf("ran privileged commands for bad input: %v", *calls)
	}
}
