package checkoutowner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"testing"
)

type chown struct {
	path     string
	uid, gid int
}

func fakeRoot(t *testing.T) *[]chown {
	t.Helper()
	var calls []chown
	geteuid = func() int { return 0 }
	lchown = func(path string, uid, gid int) error {
		calls = append(calls, chown{path, uid, gid})
		return nil
	}
	t.Cleanup(func() {
		geteuid = os.Geteuid
		lchown = os.Lchown
	})
	return &calls
}

func write(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestRepairSkipsMatchingOwner(t *testing.T) {
	calls := fakeRoot(t)
	repo := t.TempDir()
	write(t, filepath.Join(repo, "generated", "hardware.nix"))
	write(t, filepath.Join(repo, "user.config.json"))

	// Everything the test created already belongs to the test user, so a
	// root run of Repair only re-owns when the checkout owner differs.
	if os.Geteuid() == 0 {
		t.Skip("checkout owned by root: Repair is a no-op")
	}
	if err := Repair(repo); err != nil {
		t.Fatal(err)
	}
	if len(*calls) != 0 {
		t.Fatalf("re-owned files that already match: %v", *calls)
	}
}

func TestRepairReownsForeignFiles(t *testing.T) {
	calls := fakeRoot(t)
	repo := t.TempDir()
	write(t, filepath.Join(repo, "generated", "hardware.nix"))
	write(t, filepath.Join(repo, "generated", "hardware.nix.bak.20261008120000"))
	write(t, filepath.Join(repo, "notes.txt"))

	// Pretend the checkout belongs to uid/gid 4242 so every managed file
	// written by the test user counts as foreign.
	uid, gid := 4242, 4243
	foreign := walkOwner(t, repo, uid, gid)
	got := map[string]bool{}
	for _, c := range *calls {
		if c.uid != uid || c.gid != gid {
			t.Fatalf("wrong owner for %s: %d:%d", c.path, c.uid, c.gid)
		}
		got[c.path] = true
	}
	for _, path := range foreign {
		if !got[path] {
			t.Fatalf("%s not re-owned; calls %v", path, *calls)
		}
	}
	if got[filepath.Join(repo, "notes.txt")] {
		t.Fatal("re-owned an unmanaged file")
	}
}

// walkOwner runs Repair's walk with a fixed checkout owner and returns the
// managed paths it visited.
func walkOwner(t *testing.T, repo string, uid, gid int) []string {
	t.Helper()
	var visited []string
	err := walk(repo, nil, func(path string, stat *syscall.Stat_t) error {
		visited = append(visited, path)
		if int(stat.Uid) != uid || int(stat.Gid) != gid {
			return lchown(path, uid, gid)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(visited)
	want := []string{
		filepath.Join(repo, "generated"),
		filepath.Join(repo, "generated", "hardware.nix"),
		filepath.Join(repo, "generated", "hardware.nix.bak.20261008120000"),
	}
	if len(visited) != len(want) {
		t.Fatalf("visited %v, want %v", visited, want)
	}
	for i := range want {
		if visited[i] != want[i] {
			t.Fatalf("visited %v, want %v", visited, want)
		}
	}
	return visited
}

func TestRepairDoesNothingUnprivileged(t *testing.T) {
	calls := fakeRoot(t)
	geteuid = func() int { return 1000 }
	repo := t.TempDir()
	write(t, filepath.Join(repo, "generated", "state.nix"))
	if err := Repair(repo); err != nil {
		t.Fatal(err)
	}
	if len(*calls) != 0 {
		t.Fatalf("unprivileged Repair chowned: %v", *calls)
	}
}

func TestRepairRefusesEscapes(t *testing.T) {
	fakeRoot(t)
	repo := t.TempDir()
	for _, extra := range []string{"../outside", "/etc/passwd", ".."} {
		if _, err := Foreign(repo, extra); err == nil {
			t.Fatalf("accepted %q", extra)
		}
	}
}

func TestRepairRefusesSymlinkedParent(t *testing.T) {
	fakeRoot(t)
	repo := t.TempDir()
	elsewhere := t.TempDir()
	write(t, filepath.Join(elsewhere, "info", "exclude"))
	if err := os.Symlink(elsewhere, filepath.Join(repo, ".git")); err != nil {
		t.Fatal(err)
	}
	if _, err := Foreign(repo); err == nil {
		t.Fatal("followed a symlinked .git")
	}
}

func TestRepairDoesNotFollowSymlinkedManagedDir(t *testing.T) {
	calls := fakeRoot(t)
	repo := t.TempDir()
	elsewhere := t.TempDir()
	write(t, filepath.Join(elsewhere, "secret"))
	if err := os.Symlink(elsewhere, filepath.Join(repo, "generated")); err != nil {
		t.Fatal(err)
	}
	walkOwnerAny(t, repo)
	for _, c := range *calls {
		if c.path == filepath.Join(elsewhere, "secret") {
			t.Fatal("followed the generated symlink")
		}
	}
}

func walkOwnerAny(t *testing.T, repo string) {
	t.Helper()
	if err := walk(repo, nil, func(path string, _ *syscall.Stat_t) error {
		return lchown(path, 4242, 4242)
	}); err != nil {
		t.Fatal(err)
	}
}

func TestOwnerRejectsSymlinkedCheckout(t *testing.T) {
	real := t.TempDir()
	link := filepath.Join(t.TempDir(), "repo")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Owner(link); err == nil {
		t.Fatal("accepted a symlinked checkout root")
	}
}

func TestEnsureAsksSudoForForeignFiles(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("needs an unprivileged test user")
	}
	geteuid = func() int { return 1000 }
	t.Cleanup(func() { geteuid = os.Geteuid })
	repo := t.TempDir()
	write(t, filepath.Join(repo, "generated", "hardware.nix"))
	locked := filepath.Join(repo, "generated", "oddc")
	if err := os.MkdirAll(locked, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o700) })

	var got []string
	err := Ensure(context.Background(), repo, func(_ context.Context, args ...string) error {
		got = args
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	owner := fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid())
	want := []string{"chown", "-R", "-P", "-h", owner, "--", filepath.Join(repo, "generated")}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("privileged %v, want %v", got, want)
	}
}

func TestEnsureSkipsOwnedCheckout(t *testing.T) {
	repo := t.TempDir()
	write(t, filepath.Join(repo, "generated", "state.nix"))
	err := Ensure(context.Background(), repo, func(context.Context, ...string) error {
		t.Fatal("asked for privileges on an owned checkout")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
