package oddc

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSourceManifest(t *testing.T, root, rel, body string) {
	t.Helper()

	path := filepath.Join(root, "devices", rel, "device.json")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestEmbeddedSourceMetadata(t *testing.T) {
	source := EmbeddedSource{
		Root:       "/tmp/oddc",
		Repository: "openDeclarativeDeviceCollectionProject",
		Revision:   "abc123",
		Integrity:  "sha256-test",
	}

	got := source.Metadata()

	if got.Kind != "embedded" {
		t.Fatalf("Kind=%q", got.Kind)
	}
	if got.Repository != "openDeclarativeDeviceCollectionProject" {
		t.Fatalf("Repository=%q", got.Repository)
	}
	if got.Revision != "abc123" {
		t.Fatalf("Revision=%q", got.Revision)
	}
	if got.Integrity != "sha256-test" {
		t.Fatalf("Integrity=%q", got.Integrity)
	}
}

func TestResolveInheritanceOrdering(t *testing.T) {
	root := t.TempDir()

	writeSourceManifest(t, root, "laptop/common", `{
	  "schema": 1,
	  "id": "laptop/common",
	  "class": "laptop",
	  "modules": ["default.nix"],
	  "lifecycle": {"status": "supported"},
	  "validation": {}
	}`)

	writeSourceManifest(t, root, "laptop/hp", `{
	  "schema": 1,
	  "id": "laptop/hp",
	  "class": "laptop",
	  "match": {"sysVendor": ["HP"]},
	  "inherits": ["laptop/common"],
	  "modules": ["default.nix"],
	  "lifecycle": {"status": "supported"},
	  "validation": {}
	}`)

	writeSourceManifest(t, root, "laptop/hp/zbook-x2-g4", `{
	  "schema": 1,
	  "id": "laptop/hp/zbook-x2-g4",
	  "class": "laptop",
	  "match": {
	    "sysVendor": ["HP"],
	    "productName": ["HP ZBook x2 G4"],
	    "boardName": ["824C"]
	  },
	  "inherits": ["laptop/hp"],
	  "modules": ["default.nix"],
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

	want := []string{
		"laptop/common",
		"laptop/hp",
		"laptop/hp/zbook-x2-g4",
	}

	if len(resolved.Inheritance) != len(want) {
		t.Fatalf("inheritance length=%d", len(resolved.Inheritance))
	}

	for i, id := range want {
		if resolved.Inheritance[i].ID != id {
			t.Fatalf(
				"inheritance[%d]=%q, want %q",
				i,
				resolved.Inheritance[i].ID,
				id,
			)
		}
	}
}

func TestResolveRejectsMissingInheritedProfile(t *testing.T) {
	root := t.TempDir()

	writeSourceManifest(t, root, "laptop/hp", `{
	  "schema": 1,
	  "id": "laptop/hp",
	  "class": "laptop",
	  "match": {"sysVendor": ["HP"]},
	  "inherits": ["laptop/common"],
	  "lifecycle": {"status": "supported"},
	  "validation": {}
	}`)

	_, err := (EmbeddedSource{Root: root}).Resolve(Identity{
		SysVendor: "HP",
	})

	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("got %v", err)
	}
}

func TestResolveRejectsInheritanceCycle(t *testing.T) {
	root := t.TempDir()

	writeSourceManifest(t, root, "laptop/one", `{
	  "schema": 1,
	  "id": "laptop/one",
	  "class": "laptop",
	  "match": {"sysVendor": ["TEST"]},
	  "inherits": ["laptop/two"],
	  "lifecycle": {"status": "supported"},
	  "validation": {}
	}`)

	writeSourceManifest(t, root, "laptop/two", `{
	  "schema": 1,
	  "id": "laptop/two",
	  "class": "laptop",
	  "inherits": ["laptop/one"],
	  "lifecycle": {"status": "supported"},
	  "validation": {}
	}`)

	_, err := (EmbeddedSource{Root: root}).Resolve(Identity{
		SysVendor: "TEST",
	})

	if err == nil || !strings.Contains(err.Error(), "inheritance cycle") {
		t.Fatalf("got %v", err)
	}
}

func TestResolveFallsBackToClassCommonProfile(t *testing.T) {
	root := t.TempDir()

	writeSourceManifest(t, root, "laptop/common", `{
	  "schema": 1,
	  "id": "laptop/common",
	  "class": "laptop",
	  "lifecycle": {"status": "supported"},
	  "validation": {}
	}`)

	resolved, err := (EmbeddedSource{Root: root}).Resolve(Identity{
		FormFactor: "laptop",
	})
	if err != nil {
		t.Fatal(err)
	}

	if resolved.Device.ID != "laptop/common" {
		t.Fatalf("Device.ID=%q", resolved.Device.ID)
	}

	if len(resolved.Inheritance) != 1 ||
		resolved.Inheritance[0].ID != "laptop/common" {
		t.Fatalf("inheritance=%+v", resolved.Inheritance)
	}
}

func TestResolveReturnsNoMatchWithoutConcreteOrCommonProfile(t *testing.T) {
	root := t.TempDir()

	_, err := (EmbeddedSource{Root: root}).Resolve(Identity{
		FormFactor: "desktop",
	})

	if !errors.Is(err, ErrNoMatch) {
		t.Fatalf("got %v", err)
	}
}

func TestMaterializePreservesInheritanceOrder(t *testing.T) {
	root := t.TempDir()

	layers := []struct {
		rel      string
		manifest string
	}{
		{
			rel: "laptop/common",
			manifest: `{
			  "schema": 1,
			  "id": "laptop/common",
			  "class": "laptop",
			  "modules": ["default.nix"],
			  "lifecycle": {"status": "supported"},
			  "validation": {}
			}`,
		},
		{
			rel: "laptop/hp",
			manifest: `{
			  "schema": 1,
			  "id": "laptop/hp",
			  "class": "laptop",
			  "match": {"sysVendor": ["HP"]},
			  "inherits": ["laptop/common"],
			  "modules": ["default.nix"],
			  "lifecycle": {"status": "supported"},
			  "validation": {}
			}`,
		},
	}

	for _, layer := range layers {
		writeSourceManifest(t, root, layer.rel, layer.manifest)

		module := filepath.Join(root, "devices", layer.rel, "default.nix")
		if err := os.WriteFile(module, []byte("{ ... }: {}\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	source := EmbeddedSource{Root: root}
	resolved, err := source.Resolve(Identity{SysVendor: "HP"})
	if err != nil {
		t.Fatal(err)
	}

	result, err := source.Materialize(resolved, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Files) != 2 {
		t.Fatalf("materialized files=%d", len(result.Files))
	}

	if !strings.Contains(
		filepath.ToSlash(result.Files[0]),
		"laptop/common/default.nix",
	) {
		t.Fatalf("first materialized file=%q", result.Files[0])
	}

	if !strings.Contains(
		filepath.ToSlash(result.Files[1]),
		"laptop/hp/default.nix",
	) {
		t.Fatalf("second materialized file=%q", result.Files[1])
	}
}

func TestEmbeddedSourceRejectsManifestDirectoryMismatch(t *testing.T) {
	root := t.TempDir()

	writeSourceManifest(t, root, "laptop/hp", `{
	  "schema": 1,
	  "id": "laptop/dell",
	  "class": "laptop",
	  "match": {"sysVendor": ["HP"]},
	  "lifecycle": {"status": "supported"},
	  "validation": {}
	}`)

	_, err := (EmbeddedSource{Root: root}).Resolve(Identity{
		SysVendor: "HP",
	})

	if err == nil || !strings.Contains(err.Error(), "does not match directory") {
		t.Fatalf("got %v", err)
	}
}

func TestMaterializeRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	deviceDir := filepath.Join(root, "devices", "laptop", "hp")

	if err := os.MkdirAll(deviceDir, 0755); err != nil {
		t.Fatal(err)
	}

	outside := filepath.Join(root, "outside.nix")
	if err := os.WriteFile(outside, []byte("{}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(outside, filepath.Join(deviceDir, "default.nix")); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(deviceDir, "device.json"),
		[]byte(`{
		  "schema": 1,
		  "id": "laptop/hp",
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

	_, err = source.Materialize(resolved, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "escapes device directory") {
		t.Fatalf("got %v", err)
	}
}

func TestRepositoryEmbeddedSourceResolvesLaptopCommon(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	source := EmbeddedSource{
		Root:       filepath.Join(repoRoot, "oddc"),
		Repository: "openDeclarativeDeviceCollectionProject",
	}

	resolved, err := source.Resolve(Identity{
		FormFactor: "laptop",
	})
	if err != nil {
		t.Fatal(err)
	}

	if resolved.Device.ID != "laptop/common" {
		t.Fatalf("Device.ID=%q", resolved.Device.ID)
	}

	if len(resolved.Inheritance) != 1 ||
		resolved.Inheritance[0].ID != "laptop/common" {
		t.Fatalf("inheritance=%+v", resolved.Inheritance)
	}
}

func TestEquivalentSourcesResolveSameDeviceGraph(t *testing.T) {
	makeSource := func(root string) EmbeddedSource {
		t.Helper()

		writeSourceManifest(t, root, "laptop/common", `{
		  "schema": 1,
		  "id": "laptop/common",
		  "class": "laptop",
		  "modules": ["default.nix"],
		  "lifecycle": {"status": "supported"},
		  "validation": {}
		}`)

		writeSourceManifest(t, root, "laptop/hp", `{
		  "schema": 1,
		  "id": "laptop/hp",
		  "class": "laptop",
		  "match": {"sysVendor": ["HP"]},
		  "inherits": ["laptop/common"],
		  "modules": ["default.nix"],
		  "lifecycle": {"status": "supported"},
		  "validation": {}
		}`)

		for _, rel := range []string{
			"laptop/common/default.nix",
			"laptop/hp/default.nix",
		} {
			path := filepath.Join(root, "devices", filepath.FromSlash(rel))
			if err := os.WriteFile(path, []byte("{ ... }: {}\n"), 0644); err != nil {
				t.Fatal(err)
			}
		}

		return EmbeddedSource{
			Root:       root,
			Repository: "openDeclarativeDeviceCollectionProject",
			Revision:   "test-revision",
			Integrity:  "sha256-test",
		}
	}

	first := makeSource(t.TempDir())
	second := makeSource(t.TempDir())

	identity := Identity{
		FormFactor: "laptop",
		SysVendor:  "HP",
	}

	a, err := first.Resolve(identity)
	if err != nil {
		t.Fatal(err)
	}

	b, err := second.Resolve(identity)
	if err != nil {
		t.Fatal(err)
	}

	if a.Device.ID != b.Device.ID {
		t.Fatalf("device ids differ: %q != %q", a.Device.ID, b.Device.ID)
	}

	if len(a.Inheritance) != len(b.Inheritance) {
		t.Fatalf(
			"inheritance lengths differ: %d != %d",
			len(a.Inheritance),
			len(b.Inheritance),
		)
	}

	for i := range a.Inheritance {
		if a.Inheritance[i].ID != b.Inheritance[i].ID {
			t.Fatalf(
				"inheritance[%d] differs: %q != %q",
				i,
				a.Inheritance[i].ID,
				b.Inheritance[i].ID,
			)
		}
	}

	if a.Source != b.Source {
		t.Fatalf("source metadata differs: %+v != %+v", a.Source, b.Source)
	}

	fmtIDs := func(resolved Resolved) []string {
		ids := make([]string, 0, len(resolved.Inheritance))
		for _, manifest := range resolved.Inheritance {
			ids = append(ids, manifest.ID)
		}
		return ids
	}

	t.Logf("equivalent graph: %v", fmtIDs(a))
}

func TestVendorLayersRemainIsolated(t *testing.T) {
	root := t.TempDir()

	writeManifest := func(id string, body string) {
		t.Helper()

		dir := filepath.Join(root, "devices", filepath.FromSlash(id))
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(
			filepath.Join(dir, "device.json"),
			[]byte(body),
			0644,
		); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(
			filepath.Join(dir, "default.nix"),
			[]byte("{ ... }: {}\n"),
			0644,
		); err != nil {
			t.Fatal(err)
		}
	}

	writeManifest(
		"laptop/common",
		`{
  "schema": 1,
  "id": "laptop/common",
  "class": "laptop",
  "inherits": [],
  "modules": ["default.nix"],
  "lifecycle": {"status":"supported"},
  "validation": {}
}`,
	)

	writeManifest(
		"laptop/framework",
		`{
  "schema": 1,
  "id": "laptop/framework",
  "class": "laptop",
  "match": {"sysVendor":["Framework"]},
  "inherits": ["laptop/common"],
  "modules": ["default.nix"],
  "lifecycle": {"status":"supported"},
  "validation": {}
}`,
	)

	writeManifest(
		"laptop/hp",
		`{
  "schema": 1,
  "id": "laptop/hp",
  "class": "laptop",
  "match": {"sysVendor":["HP"]},
  "inherits": ["laptop/common"],
  "modules": ["default.nix"],
  "lifecycle": {"status":"supported"},
  "validation": {}
}`,
	)

	source := EmbeddedSource{
		Root: root,
	}

	hp, err := source.Resolve(Identity{
		FormFactor: "laptop",
		SysVendor:  "HP",
	})
	if err != nil {
		t.Fatal(err)
	}

	if hp.Device.ID != "laptop/hp" {
		t.Fatalf("HP resolved to %q", hp.Device.ID)
	}

	for _, layer := range hp.Inheritance {
		if layer.ID == "laptop/framework" {
			t.Fatalf("HP inherited Framework layer: %+v", hp.Inheritance)
		}
	}
}

func TestNonSelectableVendorLayerFallsBackToCommon(t *testing.T) {
	root := t.TempDir()

	writeSourceManifest(t, root, "laptop/common", `{
	  "schema": 1,
	  "id": "laptop/common",
	  "class": "laptop",
	  "lifecycle": {"status": "supported"},
	  "validation": {}
	}`)

	writeSourceManifest(t, root, "laptop/vendor", `{
	  "schema": 1,
	  "id": "laptop/vendor",
	  "class": "laptop",
	  "selectable": false,
	  "match": {"sysVendor": ["Example"]},
	  "inherits": ["laptop/common"],
	  "lifecycle": {"status": "supported"},
	  "validation": {}
	}`)

	resolved, err := (EmbeddedSource{Root: root}).Resolve(Identity{
		FormFactor:  "laptop",
		SysVendor:   "Example",
		ProductName: "Unknown Future Laptop",
	})
	if err != nil {
		t.Fatal(err)
	}

	if resolved.Device.ID != "laptop/common" {
		t.Fatalf(
			"unknown vendor model resolved to %q, want laptop/common",
			resolved.Device.ID,
		)
	}
}

func TestConcreteChildMayInheritNonSelectableVendorLayer(t *testing.T) {
	root := t.TempDir()

	writeSourceManifest(t, root, "laptop/common", `{
	  "schema": 1,
	  "id": "laptop/common",
	  "class": "laptop",
	  "lifecycle": {"status": "supported"},
	  "validation": {}
	}`)

	writeSourceManifest(t, root, "laptop/vendor", `{
	  "schema": 1,
	  "id": "laptop/vendor",
	  "class": "laptop",
	  "selectable": false,
	  "match": {"sysVendor": ["Example"]},
	  "inherits": ["laptop/common"],
	  "lifecycle": {"status": "supported"},
	  "validation": {}
	}`)

	writeSourceManifest(t, root, "laptop/vendor/model", `{
	  "schema": 1,
	  "id": "laptop/vendor/model",
	  "class": "laptop",
	  "match": {
	    "sysVendor": ["Example"],
	    "productName": ["Exact Model"],
	    "boardName": ["BOARD1"]
	  },
	  "inherits": ["laptop/vendor"],
	  "lifecycle": {"status": "experimental"},
	  "validation": {}
	}`)

	resolved, err := (EmbeddedSource{Root: root}).Resolve(Identity{
		FormFactor:  "laptop",
		SysVendor:   "Example",
		ProductName: "Exact Model",
		BoardName:   "BOARD1",
	})
	if err != nil {
		t.Fatal(err)
	}

	if resolved.Device.ID != "laptop/vendor/model" {
		t.Fatalf("resolved device=%q", resolved.Device.ID)
	}

	want := []string{
		"laptop/common",
		"laptop/vendor",
		"laptop/vendor/model",
	}

	if len(resolved.Inheritance) != len(want) {
		t.Fatalf("inheritance=%v", resolved.Inheritance)
	}

	for i, id := range want {
		if resolved.Inheritance[i].ID != id {
			t.Fatalf(
				"inheritance[%d]=%q want=%q",
				i,
				resolved.Inheritance[i].ID,
				id,
			)
		}
	}
}

func TestMultipleDMIProductNamesResolveToSameStableDeviceID(t *testing.T) {
	root := t.TempDir()

	writeSourceManifest(t, root, "laptop/common", `{
	  "schema": 1,
	  "id": "laptop/common",
	  "class": "laptop",
	  "lifecycle": {"status": "supported"},
	  "validation": {}
	}`)

	writeSourceManifest(t, root, "laptop/vendor/model-family", `{
	  "schema": 1,
	  "id": "laptop/vendor/model-family",
	  "class": "laptop",
	  "match": {
	    "sysVendor": ["Example"],
	    "productName": [
	      "Old Firmware Product Name",
	      "New Firmware Product Name"
	    ]
	  },
	  "inherits": ["laptop/common"],
	  "lifecycle": {"status": "experimental"},
	  "validation": {}
	}`)

	source := EmbeddedSource{Root: root}

	for _, product := range []string{
		"Old Firmware Product Name",
		"New Firmware Product Name",
	} {
		resolved, err := source.Resolve(Identity{
			FormFactor:  "laptop",
			SysVendor:   "Example",
			ProductName: product,
		})
		if err != nil {
			t.Fatal(err)
		}

		if resolved.Device.ID != "laptop/vendor/model-family" {
			t.Fatalf(
				"product %q resolved to %q",
				product,
				resolved.Device.ID,
			)
		}
	}
}
