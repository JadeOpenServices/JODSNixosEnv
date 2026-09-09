package oddc

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodeManifestRejectsUnknownFields(t *testing.T) {
	_, err := DecodeManifest(strings.NewReader(`{
	  "schema": 1,
	  "id": "laptop/hp/zbook-x2-g4",
	  "class": "laptop",
	  "lifecycle": {"status": "supported"},
	  "validation": {},
	  "unexpected": true
	}`))
	if err == nil {
		t.Fatal("expected unknown-field error")
	}
}

func TestValidateManifestRejectsTraversal(t *testing.T) {
	err := ValidateManifest(Manifest{
		Schema:    ManifestSchema,
		ID:        "laptop/hp/zbook-x2-g4",
		Class:     "laptop",
		Lifecycle: Lifecycle{Status: "supported"},
		Modules:   []string{"../escape.nix"},
	})
	if err == nil {
		t.Fatal("expected traversal rejection")
	}
}

func TestValidateManifestRejectsBadLifecycle(t *testing.T) {
	err := ValidateManifest(Manifest{
		Schema:    ManifestSchema,
		ID:        "laptop/hp/zbook-x2-g4",
		Class:     "laptop",
		Lifecycle: Lifecycle{Status: "unknown"},
	})
	if err == nil {
		t.Fatal("expected lifecycle rejection")
	}
}

func TestResolvePrefersMostSpecificMatch(t *testing.T) {
	root := t.TempDir()

	writeManifest := func(rel, body string) {
		t.Helper()
		path := filepath.Join(root, "devices", rel, "device.json")
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}

	writeManifest("laptop/hp", `{
	  "schema": 1,
	  "id": "laptop/hp",
	  "class": "laptop",
	  "match": {"sysVendor": ["HP"]},
	  "lifecycle": {"status": "supported"},
	  "validation": {}
	}`)

	writeManifest("laptop/hp/zbook-x2-g4", `{
	  "schema": 1,
	  "id": "laptop/hp/zbook-x2-g4",
	  "class": "laptop",
	  "match": {
	    "sysVendor": ["HP"],
	    "productName": ["HP ZBook x2 G4"],
	    "boardName": ["824C"]
	  },
	  "inherits": ["laptop/hp"],
	  "lifecycle": {"status": "supported"},
	  "validation": {}
	}`)

	resolved, err := (EmbeddedSource{Root: root}).Resolve(Identity{
		SysVendor:   "HP",
		ProductName: "HP ZBook x2 G4",
		BoardName:   "824C",
	})
	if err != nil {
		t.Fatal(err)
	}

	if resolved.Device.ID != "laptop/hp/zbook-x2-g4" {
		t.Fatalf("resolved %q", resolved.Device.ID)
	}
}

func TestResolveRejectsAmbiguousMatch(t *testing.T) {
	root := t.TempDir()

	for _, rel := range []string{"laptop/hp/one", "laptop/hp/two"} {
		path := filepath.Join(root, "devices", rel, "device.json")
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(`{
		  "schema": 1,
		  "id": "`+rel+`",
		  "class": "laptop",
		  "match": {"sysVendor": ["HP"]},
		  "lifecycle": {"status": "supported"},
		  "validation": {}
		}`), 0644); err != nil {
			t.Fatal(err)
		}
	}

	_, err := (EmbeddedSource{Root: root}).Resolve(Identity{SysVendor: "HP"})
	if !errors.Is(err, ErrAmbiguousMatch) {
		t.Fatalf("got %v", err)
	}
}

func TestResolveRejectsDuplicateIdentityOutsideCanonicalDirectory(t *testing.T) {
	root := t.TempDir()

	for _, rel := range []string{"laptop/hp/one", "laptop/hp/two"} {
		path := filepath.Join(root, "devices", rel, "device.json")
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(`{
		  "schema": 1,
		  "id": "laptop/hp/duplicate",
		  "class": "laptop",
		  "match": {"sysVendor": ["HP"]},
		  "lifecycle": {"status": "supported"},
		  "validation": {}
		}`), 0644); err != nil {
			t.Fatal(err)
		}
	}

	_, err := (EmbeddedSource{Root: root}).Resolve(Identity{SysVendor: "HP"})
	if err == nil || !strings.Contains(err.Error(), "does not match directory") {
		t.Fatalf("got %v", err)
	}
}

func TestMaterializeCopiesOnlyDeclaredModules(t *testing.T) {
	root := t.TempDir()
	deviceDir := filepath.Join(root, "devices", "laptop", "hp", "zbook-x2-g4")

	if err := os.MkdirAll(deviceDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(deviceDir, "default.nix"),
		[]byte("{}\n"),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(deviceDir, "ignored.nix"),
		[]byte("{}\n"),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(deviceDir, "device.json"),
		[]byte(`{
		  "schema": 1,
		  "id": "laptop/hp/zbook-x2-g4",
		  "class": "laptop",
		  "match": {"sysVendor": ["HP"]},
		  "modules": ["default.nix"],
		  "lifecycle": {"status": "supported"},
		  "validation": {}
		}`),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	source := EmbeddedSource{Root: root}
	resolved, err := source.Resolve(Identity{SysVendor: "HP"})
	if err != nil {
		t.Fatal(err)
	}

	target := t.TempDir()
	result, err := source.Materialize(resolved, target)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Files) != 1 {
		t.Fatalf("materialized %d files", len(result.Files))
	}

	if _, err := os.Stat(filepath.Join(
		target,
		"laptop",
		"hp",
		"zbook-x2-g4",
		"default.nix",
	)); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(
		target,
		"laptop",
		"hp",
		"zbook-x2-g4",
		"ignored.nix",
	)); !os.IsNotExist(err) {
		t.Fatalf("unexpected ignored module materialized: %v", err)
	}
}
