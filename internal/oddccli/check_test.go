package oddccli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JadeOpenServices/gjallarOS/internal/installer/oddc"
)

// upstreamRepo is a checkout with this repository's flake.lock, answered
// by a GitHub API stand-in that names rev as the branch head.
func upstreamRepo(t *testing.T, rev func(pinned string) string, status int) (string, string) {
	t.Helper()
	repo := t.TempDir()
	lock, err := os.ReadFile(realLock)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "flake.lock"), lock, 0o644); err != nil {
		t.Fatal(err)
	}
	pinned, err := oddc.LockedRevision(repo)
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/JadeOpenServices/oddc/commits/HEAD" ||
			r.Header.Get("Accept") != "application/vnd.github.sha" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(status)
		w.Write([]byte(rev(pinned)))
	}))
	t.Cleanup(server.Close)

	previous := UpstreamAPI
	UpstreamAPI = server.URL
	t.Cleanup(func() { UpstreamAPI = previous })

	return repo, pinned
}

func TestCheckUpdateReportsMovedBranch(t *testing.T) {
	moved := func(pinned string) string {
		if pinned == parentODDC {
			return lockedODDC
		}
		return parentODDC
	}
	repo, pinned := upstreamRepo(t, moved, http.StatusOK)

	var out bytes.Buffer
	CheckUpdate(repo, &out)

	want := "ODDC update available: " + short(pinned) + " -> " + short(moved(pinned))
	if !strings.Contains(out.String(), want) || !strings.Contains(out.String(), "rebuild --hardware-update") {
		t.Fatalf("output = %q, want %q and the apply hint", out.String(), want)
	}
}

func TestCheckUpdateQuietWhenCurrent(t *testing.T) {
	repo, _ := upstreamRepo(t, func(pinned string) string { return pinned }, http.StatusOK)

	var out bytes.Buffer
	CheckUpdate(repo, &out)
	if out.Len() != 0 {
		t.Fatalf("output = %q, want none", out.String())
	}
}

func TestCheckUpdateQuietOnFailure(t *testing.T) {
	repo, _ := upstreamRepo(t, func(string) string { return "rate limited" }, http.StatusForbidden)

	var out bytes.Buffer
	CheckUpdate(repo, &out)
	if out.Len() != 0 {
		t.Fatalf("output = %q, want none", out.String())
	}

	UpstreamAPI = "http://127.0.0.1:1"
	CheckUpdate(repo, &out)
	if out.Len() != 0 {
		t.Fatalf("offline output = %q, want none", out.String())
	}
}
