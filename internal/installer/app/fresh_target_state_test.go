package app

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestBindFreshTargetStateSeedsAndBindsTarget(t *testing.T) {
	calls := withPrivilegedCommand(t, nil)
	release, err := bindFreshTargetState(context.Background(), "/mnt", io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, c := range *calls {
		got = append(got, strings.Join(c, " "))
	}
	want := []string{
		"mkdir -p -- /mnt/var/lib/sbctl /var/lib/sbctl",
		"cp -a --update=none -- /var/lib/sbctl/. /mnt/var/lib/sbctl/",
		"mount --bind -- /mnt/var/lib/sbctl /var/lib/sbctl",
		"mkdir -p -- /mnt/var/lib/gjallarOS /var/lib/gjallarOS",
		"cp -a --update=none -- /var/lib/gjallarOS/. /mnt/var/lib/gjallarOS/",
		"mount --bind -- /mnt/var/lib/gjallarOS /var/lib/gjallarOS",
		"mkdir -p -- /mnt/boot /boot",
		"mount --bind -- /mnt/boot /boot",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("commands:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}

	*calls = nil
	release()
	var unmounted []string
	for _, c := range *calls {
		unmounted = append(unmounted, strings.Join(c, " "))
	}
	if want := []string{"umount -- /boot", "umount -- /var/lib/gjallarOS", "umount -- /var/lib/sbctl"}; !slices.Equal(unmounted, want) {
		t.Fatalf("release = %v, want %v", unmounted, want)
	}
}

func TestBindFreshTargetStateUndoesMountsOnFailure(t *testing.T) {
	calls := withPrivilegedCommand(t, func(args []string) ([]byte, error) {
		if args[0] == "mount" && args[len(args)-1] == "/var/lib/gjallarOS" {
			return nil, errors.New("busy")
		}
		return nil, nil
	})
	if _, err := bindFreshTargetState(context.Background(), "/mnt", io.Discard); err == nil {
		t.Fatal("bind failure was ignored")
	}
	last := strings.Join((*calls)[len(*calls)-1], " ")
	if last != "umount -- /var/lib/sbctl" {
		t.Fatalf("earlier bind mount left behind; last command %q", last)
	}
}

// The seeding copy must add live-only state (Secure Boot keys, ownership)
// without replacing what the fresh flow already staged in the target.
func TestFreshTargetStateCopyKeepsStagedFiles(t *testing.T) {
	live, target := t.TempDir(), t.TempDir()
	write := func(path, data string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(live, "passwords", "root.hash"), "live")
	write(filepath.Join(live, "secure-boot", "ownership.json"), "{}")
	write(filepath.Join(target, "passwords", "root.hash"), "staged")

	out, err := exec.Command("cp", "-a", "--update=none", "--", live+"/.", target+"/").CombinedOutput()
	if err != nil {
		t.Fatalf("cp: %v: %s", err, out)
	}
	if data, _ := os.ReadFile(filepath.Join(target, "passwords", "root.hash")); string(data) != "staged" {
		t.Fatalf("staged file replaced: %q", data)
	}
	if _, err := os.Stat(filepath.Join(target, "secure-boot", "ownership.json")); err != nil {
		t.Fatalf("live-only state not copied: %v", err)
	}
}
