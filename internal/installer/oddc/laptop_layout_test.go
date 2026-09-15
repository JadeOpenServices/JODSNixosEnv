package oddc

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

type laptopLayoutManifest struct {
	ID         string   `json:"id"`
	Class      string   `json:"class"`
	Inherits   []string `json:"inherits"`
	Modules    []string `json:"modules"`
	Selectable *bool    `json:"selectable"`
}

func layoutHasModule(modules []string, want string) bool {
	for _, module := range modules {
		if module == want {
			return true
		}
	}
	return false
}

func TestRepositoryLaptopODDCLayoutIsUniform(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "oddc", "devices", "laptop"))
	if err != nil {
		t.Fatal(err)
	}

	slug := regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || entry.Name() != "device.json" {
			return nil
		}

		dir := filepath.Dir(path)
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			return err
		}

		rel = filepath.ToSlash(rel)
		parts := strings.Split(rel, "/")

		if len(parts) > 2 {
			t.Errorf("%s has non-canonical laptop ODDC depth; want vendor or vendor/device", path)
			return nil
		}

		for _, part := range parts {
			if !slug.MatchString(part) {
				t.Errorf("%s contains non-kebab-case path component %q", path, part)
			}
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		var manifest laptopLayoutManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			return err
		}

		wantID := "laptop/" + rel
		if manifest.ID != wantID {
			t.Errorf("%s id=%q want=%q", path, manifest.ID, wantID)
		}

		if manifest.Class != "laptop" {
			t.Errorf("%s class=%q want=laptop", path, manifest.Class)
		}

		if !layoutHasModule(manifest.Modules, "default.nix") {
			t.Errorf("%s must list default.nix in modules", path)
		}

		switch {
		case rel == "common":
			if len(manifest.Inherits) != 0 {
				t.Errorf("%s must not inherit another device layer", path)
			}

		case len(parts) == 1:
			want := []string{"laptop/common"}
			if !reflect.DeepEqual(manifest.Inherits, want) {
				t.Errorf("%s inherits=%v want=%v", path, manifest.Inherits, want)
			}
			if manifest.Selectable == nil || *manifest.Selectable {
				t.Errorf("%s vendor layer must set selectable=false", path)
			}

		case len(parts) == 2:
			want := []string{"laptop/" + parts[0]}
			if !reflect.DeepEqual(manifest.Inherits, want) {
				t.Errorf("%s inherits=%v want=%v", path, manifest.Inherits, want)
			}
		}

		if _, err := os.Stat(filepath.Join(dir, "default.nix")); err != nil {
			t.Errorf("%s has device.json but no default.nix", dir)
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
