package flakesource

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, path, data string, perm os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), perm); err != nil {
		t.Fatal(err)
	}
}

func TestStageKeepsFlakeInputsAndDropsHistoryAndScratch(t *testing.T) {
	repo := t.TempDir()
	write(t, filepath.Join(repo, "flake.nix"), "{}", 0o644)
	write(t, filepath.Join(repo, "generated", "state.nix"), "{}", 0o644) // gitignored, required
	write(t, filepath.Join(repo, "scripts", "run.sh"), "#!/bin/sh", 0o755)
	write(t, filepath.Join(repo, ".git", "objects", "pack", "big.pack"), "history", 0o644)
	write(t, filepath.Join(repo, "pkgs", "monique", ".git"), "gitdir: ../../.git/modules/monique", 0o644)
	write(t, filepath.Join(repo, "pkgs", "monique", "default.nix"), "{}", 0o644)
	write(t, filepath.Join(repo, ".vm", "disk.qcow2"), "vm", 0o644)
	write(t, filepath.Join(repo, "system", "result", "keep.nix"), "{}", 0o644) // nested "result" is source
	if err := os.Symlink("/nix/store/x-system", filepath.Join(repo, "result")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("run.sh", filepath.Join(repo, "scripts", "alias")); err != nil {
		t.Fatal(err)
	}

	src, err := Stage(repo, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()

	for _, keep := range []string{"flake.nix", "generated/state.nix", "pkgs/monique/default.nix", "system/result/keep.nix"} {
		if _, err := os.Stat(filepath.Join(src.Dir, keep)); err != nil {
			t.Errorf("missing %s: %v", keep, err)
		}
	}
	for _, drop := range []string{".git", "pkgs/monique/.git", ".vm", "result"} {
		if _, err := os.Lstat(filepath.Join(src.Dir, drop)); !os.IsNotExist(err) {
			t.Errorf("%s was staged: %v", drop, err)
		}
	}
	if info, err := os.Stat(filepath.Join(src.Dir, "scripts", "run.sh")); err != nil || info.Mode().Perm()&0o100 == 0 {
		t.Errorf("executable bit lost: %v %v", info, err)
	}
	if link, err := os.Readlink(filepath.Join(src.Dir, "scripts", "alias")); err != nil || link != "run.sh" {
		t.Errorf("symlink not preserved: %q %v", link, err)
	}
	if got := src.Ref("gjallar-test"); got != "path:"+src.Dir+"#gjallar-test" {
		t.Errorf("Ref = %q", got)
	}

	parent := filepath.Dir(src.Dir)
	if err := src.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(parent); !os.IsNotExist(err) {
		t.Errorf("staging directory left behind: %v", err)
	}
}

func TestStageRejectsNonFlakeDirectory(t *testing.T) {
	if _, err := Stage(t.TempDir(), t.TempDir()); err == nil {
		t.Fatal("staged a directory without flake.nix")
	}
}
