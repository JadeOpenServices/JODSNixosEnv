package secrets

import (
	"os"
	"os/user"
	"path/filepath"
	"testing"
)

func TestMissingDirsBelowHome(t *testing.T) {
	home := t.TempDir()
	if err := os.Mkdir(filepath.Join(home, ".config"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := missingDirs(home, filepath.Join(home, ".config", "sops", "age"))
	want := []string{
		filepath.Join(home, ".config", "sops"),
		filepath.Join(home, ".config", "sops", "age"),
	}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("missingDirs = %v, want %v", got, want)
	}
}

func TestGiveToAccountOnlyAsRoot(t *testing.T) {
	var chowned []string
	secretsLchown = func(path string, uid, gid int) error {
		if uid != 1000 || gid != 100 {
			t.Fatalf("%s to %d:%d", path, uid, gid)
		}
		chowned = append(chowned, path)
		return nil
	}
	t.Cleanup(func() {
		secretsEUID = os.Geteuid
		secretsLchown = os.Lchown
	})
	account := &user.User{Uid: "1000", Gid: "100", Username: "tester"}

	secretsEUID = func() int { return 1000 }
	if err := giveToAccount(account, "/home/tester/.config/sops/age/keys.txt"); err != nil || len(chowned) != 0 {
		t.Fatalf("unprivileged chown: %v %v", err, chowned)
	}
	secretsEUID = func() int { return 0 }
	if err := giveToAccount(account, "a", "b"); err != nil || len(chowned) != 2 {
		t.Fatalf("root chown: %v %v", err, chowned)
	}
}
