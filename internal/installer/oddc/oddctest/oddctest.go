// Package oddctest gives tests the real ODDC catalog this build depends
// on, one model at a time, fetched as the installer fetches it.
package oddctest

import (
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	portable "github.com/JadeOpenServices/oddc/pkg/oddc"
)

// Answer is one catalog model as the installer receives it.
type Answer struct {
	Model    string
	Root     string
	Identity portable.MachineIdentity
}

// Catalog returns the source tree of the ODDC module in go.mod.
func Catalog(t testing.TB) string {
	t.Helper()

	out, err := exec.Command(
		"go", "list", "-m", "-f", "{{.Dir}}", "github.com/JadeOpenServices/oddc",
	).Output()
	if err != nil {
		t.Fatalf("locate ODDC module: %v", err)
	}

	return strings.TrimSpace(string(out))
}

// Answers fetches every catalog model into its own answer directory.
func Answers(t testing.TB) []Answer {
	t.Helper()

	catalog := Catalog(t)
	registry, err := portable.LoadRegistry(catalog)
	if err != nil {
		t.Fatal(err)
	}

	var models []string
	for id, entity := range registry.Entities {
		if entity.Kind == "DeviceModel" {
			models = append(models, id)
		}
	}
	sort.Strings(models)
	if len(models) == 0 {
		t.Fatal("ODDC catalog has no models")
	}

	answers := make([]Answer, 0, len(models))
	for _, model := range models {
		identity, err := registry.ModelIdentity(model)
		if err != nil {
			t.Fatal(err)
		}

		root := filepath.Join(t.TempDir(), "oddc")
		if err := portable.FetchModel(portable.DirSource{Root: catalog}, model, root); err != nil {
			t.Fatal(err)
		}

		answers = append(answers, Answer{Model: model, Root: root, Identity: identity})
	}

	return answers
}
