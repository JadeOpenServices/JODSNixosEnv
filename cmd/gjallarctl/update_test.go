package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type updateFixture struct {
	t       *testing.T
	remote  string
	work    string
	clone   string
	key     string
	rebuilt int
}

func mustRun(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func newUpdateFixture(t *testing.T, channel string) *updateFixture {
	keygen, err := exec.LookPath("ssh-keygen")
	if err != nil {
		t.Skip("ssh-keygen not available")
	}
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_AUTHOR_NAME", "Release")
	t.Setenv("GIT_AUTHOR_EMAIL", "release@gjallar.invalid")
	t.Setenv("GIT_COMMITTER_NAME", "Release")
	t.Setenv("GIT_COMMITTER_EMAIL", "release@gjallar.invalid")

	f := &updateFixture{
		t:      t,
		remote: filepath.Join(root, "remote.git"),
		work:   filepath.Join(root, "work"),
		clone:  filepath.Join(root, "clone"),
		key:    filepath.Join(root, "release"),
	}
	mustRun(t, root, keygen, "-q", "-t", "ed25519", "-N", "", "-C", "release", "-f", f.key)
	pub, err := os.ReadFile(f.key + ".pub")
	if err != nil {
		t.Fatal(err)
	}
	signers := filepath.Join(root, "allowed_signers")
	writeFile(t, signers, "# GjallarOS release keys\nrelease@gjallar.invalid namespaces=\"git\" "+string(pub))

	mustRun(t, root, "git", "init", "-q", "--bare", "-b", "main", f.remote)
	mustRun(t, root, "git", "init", "-q", "-b", "main", f.work)
	writeFile(t, filepath.Join(f.work, "flake.lock"), "release lock\n")
	f.commit("base")
	f.push()
	mustRun(t, root, "git", "clone", "-q", f.remote, f.clone)

	cfg := filepath.Join(root, "update.json")
	data, _ := json.Marshal(updateConfig{
		Channel: channel, Remote: f.remote, Release: "26.05",
		AllowedSigners: signers, SSHKeygen: keygen,
	})
	writeFile(t, cfg, string(data))

	oldPath, oldRebuild := updateConfigPath, updateRebuild
	updateConfigPath = cfg
	updateRebuild = func(io.Writer, io.Writer) int { f.rebuilt++; return 0 }
	t.Cleanup(func() { updateConfigPath, updateRebuild = oldPath, oldRebuild })
	t.Setenv("GJALLAROS_REPO", f.clone)
	return f
}

func writeFile(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func (f *updateFixture) sign() []string {
	return []string{"-c", "gpg.format=ssh", "-c", "user.signingkey=" + f.key}
}

func (f *updateFixture) commit(msg string) string {
	writeFile(f.t, filepath.Join(f.work, "file"), msg)
	mustRun(f.t, f.work, "git", "add", "-A")
	mustRun(f.t, f.work, "git", append(f.sign(), "commit", "-q", "-S", "-m", msg)...)
	return mustRun(f.t, f.work, "git", "rev-parse", "HEAD")
}

func (f *updateFixture) unsignedCommit(msg string) string {
	writeFile(f.t, filepath.Join(f.work, "file"), msg)
	mustRun(f.t, f.work, "git", "add", "-A")
	mustRun(f.t, f.work, "git", "commit", "-q", "-m", msg)
	return mustRun(f.t, f.work, "git", "rev-parse", "HEAD")
}

func (f *updateFixture) tag(name string, signed bool) {
	args := []string{"tag", "-a", "-m", name, name}
	if signed {
		args = append(f.sign(), "tag", "-s", "-m", name, name)
	}
	mustRun(f.t, f.work, "git", args...)
}

func (f *updateFixture) push() {
	mustRun(f.t, f.work, "git", "push", "-q", "--force", "--tags", f.remote, "main")
}

func (f *updateFixture) head() string {
	return mustRun(f.t, f.clone, "git", "rev-parse", "HEAD")
}

func (f *updateFixture) update(args ...string) (int, string) {
	var stdout, stderr bytes.Buffer
	code := runUpdate(args, &stdout, &stderr)
	return code, stdout.String() + stderr.String()
}

func TestUpdateStableMovesToNewestSignedTag(t *testing.T) {
	f := newUpdateFixture(t, "stable")
	f.commit("one")
	f.tag("v26.05.1", true)
	want := f.commit("two")
	f.tag("v26.05.2", true)
	f.commit("unreleased")
	f.push()

	if code, out := f.update("--check"); code != 0 || !strings.Contains(out, "Update available: v26.05.2") {
		t.Fatalf("check: %d %s", code, out)
	}
	if f.rebuilt != 0 {
		t.Fatal("--check rebuilt")
	}
	if code, out := f.update(); code != 0 {
		t.Fatalf("update: %d %s", code, out)
	}
	if f.head() != want || f.rebuilt != 1 {
		t.Fatalf("head %s rebuilt %d, want %s once", f.head(), f.rebuilt, want)
	}
	if code, out := f.update(); code != 0 || !strings.Contains(out, "up to date (v26.05.2)") {
		t.Fatalf("second update: %d %s", code, out)
	}
}

func TestUpdateRefusesUnverifiedReleases(t *testing.T) {
	for name, publish := range map[string]func(*updateFixture){
		"unsigned tag": func(f *updateFixture) {
			f.commit("one")
			f.tag("v26.05.1", false)
		},
		"foreign key": func(f *updateFixture) {
			f.commit("one")
			other := filepath.Join(filepath.Dir(f.key), "other")
			mustRun(f.t, f.work, "ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", other)
			mustRun(f.t, f.work, "git", "-c", "gpg.format=ssh", "-c", "user.signingkey="+other,
				"tag", "-s", "-m", "v26.05.1", "v26.05.1")
		},
		"renamed signed tag": func(f *updateFixture) {
			f.commit("one")
			f.tag("v26.05.1", true)
			object := mustRun(f.t, f.work, "git", "rev-parse", "v26.05.1")
			mustRun(f.t, f.work, "git", "update-ref", "refs/tags/v26.05.9", object)
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := newUpdateFixture(t, "stable")
			before := f.head()
			publish(f)
			f.push()
			if code, out := f.update(); code == 0 || !strings.Contains(out, "not updating") {
				t.Fatalf("update: %d %s", code, out)
			}
			if f.head() != before || f.rebuilt != 0 {
				t.Fatal("unverified release was applied")
			}
		})
	}
}

func TestUpdateMainChannelNeedsSignedHead(t *testing.T) {
	f := newUpdateFixture(t, "main")
	f.unsignedCommit("unsigned")
	f.push()
	before := f.head()
	if code, out := f.update(); code == 0 || !strings.Contains(out, "not signed") {
		t.Fatalf("unsigned main: %d %s", code, out)
	}
	if f.head() != before {
		t.Fatal("unsigned main was applied")
	}
	want := f.commit("signed")
	f.push()
	if code, out := f.update(); code != 0 || f.head() != want {
		t.Fatalf("signed main: %d %s", code, out)
	}
}

func TestUpdateKeepsLocalWork(t *testing.T) {
	f := newUpdateFixture(t, "stable")
	f.commit("one")
	f.tag("v26.05.1", true)
	f.push()

	writeFile(t, filepath.Join(f.clone, "file"), "edited")
	if code, out := f.update(); code == 0 || !strings.Contains(out, "uncommitted changes") {
		t.Fatalf("dirty: %d %s", code, out)
	}
	mustRun(t, f.clone, "git", "checkout", "-q", "--", "file")

	mustRun(t, f.clone, "git", "commit", "-q", "--allow-empty", "-m", "local")
	if code, out := f.update(); code == 0 || !strings.Contains(out, "commits that are not in v26.05.1") {
		t.Fatalf("diverged: %d %s", code, out)
	}
	if f.rebuilt != 0 {
		t.Fatal("rebuilt despite refusal")
	}
}

func TestUpdateNeedsSigningKeys(t *testing.T) {
	f := newUpdateFixture(t, "stable")
	var cfg updateConfig
	data, _ := os.ReadFile(updateConfigPath)
	_ = json.Unmarshal(data, &cfg)
	writeFile(t, cfg.AllowedSigners, "# none yet\n")
	if code, out := f.update(); code == 0 || !strings.Contains(out, "no release signing keys") {
		t.Fatalf("no keys: %d %s", code, out)
	}
}

func TestUpdateReplacesMovedFlakeLock(t *testing.T) {
	f := newUpdateFixture(t, "stable")
	want := f.commit("one")
	f.tag("v26.05.1", true)
	f.push()

	lock := filepath.Join(f.clone, "flake.lock")
	writeFile(t, lock, "oddc moved\n")
	if code, out := f.update(); code != 0 || !strings.Contains(out, "flake.lock changes replaced") {
		t.Fatalf("update: %d %s", code, out)
	}
	data, _ := os.ReadFile(lock)
	if f.head() != want || string(data) != "release lock\n" {
		t.Fatalf("head %s lock %q", f.head(), data)
	}
}
