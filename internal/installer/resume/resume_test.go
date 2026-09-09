package resume

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func writeTransaction(t *testing.T, tx Transaction) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "transaction.json")

	data, err := json.Marshal(tx)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}

	return path
}

func TestLoad(t *testing.T) {
	wantArgs := []string{
		"-repo",
		"/home/test/gjallarOS",
		"-accept-existing",
	}

	path := writeTransaction(t, Transaction{
		Schema:          Schema,
		State:           "waiting-for-release-reboot",
		Reason:          "nixos-release-alignment",
		ExpectedRelease: "26.05",
		Repo:            "/home/test/gjallarOS",
		Args:            wantArgs,
	})

	tx, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if tx.ExpectedRelease != "26.05" {
		t.Fatalf(
			"expected release %q, want 26.05",
			tx.ExpectedRelease,
		)
	}

	if !reflect.DeepEqual(tx.Args, wantArgs) {
		t.Fatalf("args %#v, want %#v", tx.Args, wantArgs)
	}
}

func TestLoadRejectsFailedState(t *testing.T) {
	path := writeTransaction(t, Transaction{
		Schema:          Schema,
		State:           "failed",
		ExpectedRelease: "26.05",
		Repo:            "/home/test/gjallarOS",
	})

	if _, err := Load(path); err == nil {
		t.Fatal("expected failed transaction to be rejected")
	}
}

func TestLoadRejectsRelativeRepo(t *testing.T) {
	path := writeTransaction(t, Transaction{
		Schema:          Schema,
		State:           "waiting-for-release-reboot",
		ExpectedRelease: "26.05",
		Repo:            "relative/repo",
	})

	if _, err := Load(path); err == nil {
		t.Fatal("expected relative repository to be rejected")
	}
}
