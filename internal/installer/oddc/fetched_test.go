package oddc

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	portable "github.com/JadeOpenServices/oddc/pkg/oddc"

	"github.com/bakanura/gjallarOS/internal/installer/oddc/oddctest"
)

// offline fails every request, as a rebuild without network does.
var offline = &http.Client{Transport: roundTripper(func(*http.Request) (*http.Response, error) {
	return nil, errors.New("offline")
})}

type roundTripper func(*http.Request) (*http.Response, error)

func (fn roundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestRefreshWithoutAnswerDoesNothing(t *testing.T) {
	root := filepath.Join(t.TempDir(), "oddc")

	before, err := Refresh(root, "4e311931ed5fac2dca79b99d01e50e620624afa0", offline)
	if err != nil || before != "" {
		t.Fatalf("Refresh = %q, %v; want no answer, no error", before, err)
	}
	if _, err := os.Stat(root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Refresh created %s", root)
	}
}

func TestRefreshAtSameRevisionDoesNotAsk(t *testing.T) {
	for _, answer := range oddctest.Answers(t) {
		current := portable.DirSource{Root: answer.Root}.Revision()

		before, err := Refresh(answer.Root, current, offline)
		if err != nil || before != current {
			t.Fatalf("%s: Refresh = %q, %v; want %q without asking", answer.Model, before, err, current)
		}
	}
}

func TestRefreshOfflineKeepsAnswer(t *testing.T) {
	for _, answer := range oddctest.Answers(t) {
		current := portable.DirSource{Root: answer.Root}.Revision()
		rev := strings.Repeat("0", 40)

		before, err := Refresh(answer.Root, rev, offline)
		if err == nil {
			t.Fatalf("%s: Refresh offline succeeded", answer.Model)
		}
		if before != current {
			t.Errorf("%s: before = %q, want %q", answer.Model, before, current)
		}
		registry, err := portable.LoadRegistry(answer.Root)
		if err != nil {
			t.Fatalf("%s: answer lost: %v", answer.Model, err)
		}
		if _, ok := registry.Entities[answer.Model]; !ok {
			t.Fatalf("%s: answer no longer holds its model", answer.Model)
		}
	}
}

func TestRefreshRejectsWholeCatalog(t *testing.T) {
	catalog := t.TempDir()
	if err := os.CopyFS(catalog, os.DirFS(oddctest.Catalog(t))); err != nil {
		t.Fatal(err)
	}

	if _, err := Refresh(catalog, strings.Repeat("0", 40), offline); err == nil ||
		!strings.Contains(err.Error(), "want 1") {
		t.Fatalf("Refresh of a whole catalog: %v", err)
	}
}

func TestLockedRevisionReadsRepositoryLock(t *testing.T) {
	rev, err := LockedRevision("../../..")
	if err != nil {
		t.Fatal(err)
	}
	if len(rev) != 40 || strings.Trim(rev, "0123456789abcdef") != "" {
		t.Fatalf("LockedRevision = %q, want a commit", rev)
	}
}

func TestLockedRevisionWithoutODDCInput(t *testing.T) {
	data, err := os.ReadFile("../../../flake.lock")
	if err != nil {
		t.Fatal(err)
	}
	repo := t.TempDir()
	broken := strings.Replace(string(data), `"oddc": "oddc"`, `"oddc-gone": "oddc"`, 1)
	if broken == string(data) {
		t.Fatal(`flake.lock has no "oddc": "oddc" root input`)
	}
	if err := os.WriteFile(filepath.Join(repo, "flake.lock"), []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := LockedRevision(repo); err == nil || !strings.Contains(err.Error(), "no oddc input") {
		t.Fatalf("LockedRevision = %v, want no oddc input", err)
	}
}
