package usbtrust

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSecureStateDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state")

	if err := EnsureSecureStateDirectory(dir); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}

	if got := info.Mode().Perm(); got != StateDirectoryMode {
		t.Fatalf(
			"state directory mode = %04o, want %04o",
			got,
			StateDirectoryMode,
		)
	}
}

func TestRejectsWritableStateDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state")

	if err := os.Mkdir(dir, 0770); err != nil {
		t.Fatal(err)
	}

	if err := EnsureSecureStateDirectory(dir); err == nil {
		t.Fatal("accepted insecure USB trust directory")
	}
}

func TestRejectsSymlinkStateDirectory(t *testing.T) {
	root := t.TempDir()

	target := filepath.Join(root, "real")
	link := filepath.Join(root, "state")

	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	if err := EnsureSecureStateDirectory(link); err == nil {
		t.Fatal("accepted symlink USB trust directory")
	}
}

func TestSecureStateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trust.json")

	if err := os.WriteFile(path, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := ValidateSecureStateFile(path); err != nil {
		t.Fatal(err)
	}
}

func TestRejectsReadableByOtherUsers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trust.json")

	if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := ValidateSecureStateFile(path); err == nil {
		t.Fatal("accepted world-readable USB trust state")
	}
}

func TestRejectsSymlinkStateFile(t *testing.T) {
	root := t.TempDir()

	target := filepath.Join(root, "real")
	link := filepath.Join(root, "trust.json")

	if err := os.WriteFile(target, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	if err := ValidateSecureStateFile(link); err == nil {
		t.Fatal("accepted symlink USB trust state")
	}
}
