package oddcdraft

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	portable "github.com/JadeOpenServices/oddc/pkg/oddc"

	"github.com/JadeOpenServices/gjallarOS/internal/installer/oddc/oddctest"
)

// A machine ODDC matches gets no draft: every real catalog model, as it
// reports itself.
func TestWriteSkipsMatchedModels(t *testing.T) {
	catalog := oddctest.Catalog(t)
	registry, err := portable.LoadRegistry(catalog)
	if err != nil {
		t.Fatal(err)
	}

	for _, answer := range oddctest.Answers(t) {
		facts, err := registry.ModelFacts(answer.Model)
		if err != nil {
			t.Fatal(err)
		}
		dir := filepath.Join(t.TempDir(), "draft")
		if _, err := Write(catalog, facts, dir); !errors.Is(err, ErrMatched) {
			t.Fatalf("%s: err = %v, want ErrMatched", answer.Model, err)
		}
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Fatalf("%s: draft directory written", answer.Model)
		}
	}
}

func TestWriteNeedsCatalog(t *testing.T) {
	if _, err := Write("", portable.Facts{}, t.TempDir()); err == nil {
		t.Fatal("no error without a catalog")
	}
}
