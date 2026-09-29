package installstate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFreshBaselineAndRerun(t *testing.T) {
	repo := t.TempDir()
	if err := Ensure(context.Background(), repo, "", "tester", "26.05"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(repo, "generated", "install-state.nix")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"nixosStateVersion", "homeManagerStateVersion"} {
		if !strings.Contains(string(before), field+` = "26.05";`) {
			t.Fatalf("missing %s in %s", field, before)
		}
	}
	if err := Ensure(context.Background(), repo, "/nonexistent/configuration.nix", "tester", "26.11"); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(path)
	if string(after) != string(before) {
		t.Fatal("rerun advanced compatibility baselines")
	}
}

func TestExistingVersionsRemainIndependentOfRelease(t *testing.T) {
	repo := t.TempDir()
	if err := ensure(repo, func() (Versions, error) {
		return Versions{NixOS: "23.11", Home: "24.05"}, nil
	}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(repo, "generated", "install-state.nix"))
	if !strings.Contains(string(data), `nixosStateVersion = "23.11"`) ||
		!strings.Contains(string(data), `homeManagerStateVersion = "24.05"`) {
		t.Fatalf("historical versions lost: %s", data)
	}
}

func TestFailedDiscoveryDoesNotPublishState(t *testing.T) {
	for _, tc := range []struct {
		name     string
		versions Versions
		err      error
	}{
		{"evaluation failure", Versions{}, fmt.Errorf("configuration unavailable")},
		{"invalid version", Versions{NixOS: "26.05", Home: ""}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := t.TempDir()
			if err := ensure(repo, func() (Versions, error) { return tc.versions, tc.err }); err == nil {
				t.Fatal("accepted missing compatibility state")
			}
			if _, err := os.Stat(filepath.Join(repo, "generated", "install-state.nix")); !os.IsNotExist(err) {
				t.Fatal("published state after discovery failure")
			}
		})
	}
}
