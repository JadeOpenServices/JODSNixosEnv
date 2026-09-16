package deviceprofilecache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bakanura/gjallarOS/internal/installer/oddc"
	portable "github.com/bakanura/gjallarOS/pkg/oddc"
)

func writeCanonicalTestModel(
	t *testing.T,
	root string,
) {
	t.Helper()

	path := filepath.Join(
		root,
		"catalog",
		"entities",
		"models",
		"test",
		"test-laptop.json",
	)

	if err := os.MkdirAll(
		filepath.Dir(path),
		0755,
	); err != nil {
		t.Fatal(err)
	}

	body := `{
	  "apiVersion": "oddc.openjade.de/v2",
	  "kind": "DeviceModel",
	  "metadata": {
	    "id": "model/test/test-laptop",
	    "name": "Test Laptop"
	  },
	  "data": {
	    "class": {
	      "formFactor": "laptop"
	    },
	    "identity": {
	      "dmi": {
	        "systemVendor": {
	          "hp": "HP"
	        },
	        "productName": {
	          "test-laptop": "Test Laptop"
	        }
	      }
	    }
	  }
	}`

	if err := os.WriteFile(
		path,
		[]byte(body),
		0644,
	); err != nil {
		t.Fatal(err)
	}
}

func testResolved(
	t *testing.T,
) (
	oddc.EmbeddedSource,
	oddc.Identity,
	oddc.Resolved,
) {
	t.Helper()

	root := t.TempDir()
	writeCanonicalTestModel(t, root)

	source := oddc.EmbeddedSource{
		Root:       root,
		Repository: "embedded:oddc",
		Revision:   "oddc-test-revision",
		Integrity:  "sha256-source-test",
	}

	identity := oddc.Identity{
		FormFactor:     "laptop",
		SysVendor:      "HP",
		ProductName:    "Test Laptop",
		ProductVersion: "rev-a",
		BoardVendor:    "HP",
		BoardName:      "board-a",
		BoardVersion:   "firmware-a",
	}

	registry, err := portable.LoadRegistry(root)
	if err != nil {
		t.Fatal(err)
	}

	canonical, err := registry.ResolveModel(
		"model/test/test-laptop",
		nil,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	resolved := oddc.Resolved{
		ModelID:   "model/test/test-laptop",
		Canonical: canonical,
	}

	return source, identity, resolved
}

func materializedTestCapsule(t *testing.T) string {
	t.Helper()

	source, identity, resolved := testResolved(t)
	destination := filepath.Join(t.TempDir(), "device-profile")

	if _, err := Materialize(MaterializeInput{
		Destination:       destination,
		Identity:          identity,
		Resolved:          resolved,
		Source:            source,
		GjallarOSRevision: "gjallar-test-revision",
		NixOSRelease:      "26.05",
		GenerationReason:  "initial",
	}); err != nil {
		t.Fatal(err)
	}

	return destination
}

func TestMaterializeAndVerify(t *testing.T) {
	source, identity, resolved := testResolved(t)

	destination := filepath.Join(t.TempDir(), "device-profile")

	capsule, err := Materialize(MaterializeInput{
		Destination:       destination,
		Identity:          identity,
		Resolved:          resolved,
		Source:            source,
		GjallarOSRevision: "gjallar-test-revision",
		NixOSRelease:      "26.05",
		GenerationReason:  "initial",
	})
	if err != nil {
		t.Fatal(err)
	}

	if capsule.Manifest.Schema != Schema {
		t.Fatalf("schema=%d", capsule.Manifest.Schema)
	}

	if capsule.Manifest.ModelID != "model/test/test-laptop" {
		t.Fatalf("model id=%q", capsule.Manifest.ModelID)
	}

	wantPaths := map[string]bool{
		"oddc/catalog/entities/models/test/test-laptop.json": false,
		"oddc/resolved.json": false,
	}

	for _, file := range capsule.Manifest.Files {
		if _, ok := wantPaths[file.Path]; ok {
			wantPaths[file.Path] = true
		}

		if len(file.SHA256) != 64 {
			t.Fatalf(
				"invalid sha256 for %q: %q",
				file.Path,
				file.SHA256,
			)
		}
	}

	for wanted, found := range wantPaths {
		if !found {
			t.Fatalf(
				"canonical capsule missing %q",
				wanted,
			)
		}
	}

	verified, err := Verify(destination)
	if err != nil {
		t.Fatal(err)
	}
	if verified.Source.GjallarOSRevision != "gjallar-test-revision" {
		t.Fatalf(
			"GjallarOS revision=%q",
			verified.Source.GjallarOSRevision,
		)
	}
	if verified.Source.ODDCRevision != "oddc-test-revision" {
		t.Fatalf("ODDC revision=%q", verified.Source.ODDCRevision)
	}
	if verified.Source.NixOSRelease != "26.05" {
		t.Fatalf("NixOS release=%q", verified.Source.NixOSRelease)
	}
}

func TestVerifyDetectsModifiedCanonicalFile(t *testing.T) {
	source, identity, resolved := testResolved(t)

	destination := filepath.Join(t.TempDir(), "device-profile")

	capsule, err := Materialize(MaterializeInput{
		Destination:       destination,
		Identity:          identity,
		Resolved:          resolved,
		Source:            source,
		GjallarOSRevision: "gjallar-test-revision",
		NixOSRelease:      "26.05",
	})
	if err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(
		destination,
		filepath.FromSlash(capsule.Manifest.Files[0].Path),
	)
	if err := os.WriteFile(target, []byte("tampered\n"), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := Verify(destination); err == nil ||
		!strings.Contains(err.Error(), "integrity mismatch") {
		t.Fatalf("verification error=%v", err)
	}
}

func TestNeedsRebindIgnoresFirmwareRevisionOnlyChange(t *testing.T) {
	source, identity, resolved := testResolved(t)

	destination := filepath.Join(t.TempDir(), "device-profile")

	capsule, err := Materialize(MaterializeInput{
		Destination:       destination,
		Identity:          identity,
		Resolved:          resolved,
		Source:            source,
		GjallarOSRevision: "gjallar-test-revision",
		NixOSRelease:      "26.05",
	})
	if err != nil {
		t.Fatal(err)
	}

	current := identity
	current.ProductVersion = "rev-b"
	current.BoardVersion = "firmware-b"

	if NeedsRebind(capsule, current, resolved) {
		t.Fatal("firmware/version-only DMI change triggered device rebind")
	}
}

func TestNeedsRebindDetectsReplacementHardware(t *testing.T) {
	source, identity, resolved := testResolved(t)

	destination := filepath.Join(t.TempDir(), "device-profile")

	capsule, err := Materialize(MaterializeInput{
		Destination:       destination,
		Identity:          identity,
		Resolved:          resolved,
		Source:            source,
		GjallarOSRevision: "gjallar-test-revision",
		NixOSRelease:      "26.05",
	})
	if err != nil {
		t.Fatal(err)
	}

	current := identity
	current.SysVendor = "Framework"
	current.ProductName = "Framework Laptop 13"
	current.BoardVendor = "Framework"
	current.BoardName = "FRANMDCP"

	if !NeedsRebind(capsule, current, resolved) {
		t.Fatal("replacement machine was not detected")
	}
}

func TestNeedsRebindDetectsResolvedModelChange(t *testing.T) {
	source, identity, resolved := testResolved(t)

	destination := filepath.Join(t.TempDir(), "device-profile")

	capsule, err := Materialize(MaterializeInput{
		Destination:       destination,
		Identity:          identity,
		Resolved:          resolved,
		Source:            source,
		GjallarOSRevision: "gjallar-test-revision",
		NixOSRelease:      "26.05",
	})
	if err != nil {
		t.Fatal(err)
	}

	changed := resolved
	changed.ModelID = "model/test/replacement"

	if !NeedsRebind(capsule, identity, changed) {
		t.Fatal("resolved device profile change was not detected")
	}
}

func TestMaterializeReplacesOldCapsuleWithoutRetainingOldModules(t *testing.T) {
	source, identity, resolved := testResolved(t)

	parent := t.TempDir()
	destination := filepath.Join(parent, "device-profile")

	if err := os.MkdirAll(
		filepath.Join(destination, "modules", "stale"),
		0700,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(destination, "modules", "stale", "old.nix"),
		[]byte("stale\n"),
		0600,
	); err != nil {
		t.Fatal(err)
	}

	_, err := Materialize(MaterializeInput{
		Destination:       destination,
		Identity:          identity,
		Resolved:          resolved,
		Source:            source,
		GjallarOSRevision: "gjallar-test-revision",
		NixOSRelease:      "26.05",
		GenerationReason:  "hardware-rebind",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(
		filepath.Join(destination, "modules", "stale", "old.nix"),
	); !os.IsNotExist(err) {
		t.Fatalf("stale module survived capsule replacement: %v", err)
	}
}

func TestVerifyRejectsUnlistedCanonicalFile(t *testing.T) {
	source, identity, resolved := testResolved(t)

	destination := filepath.Join(t.TempDir(), "device-profile")

	_, err := Materialize(MaterializeInput{
		Destination:       destination,
		Identity:          identity,
		Resolved:          resolved,
		Source:            source,
		GjallarOSRevision: "gjallar-test-revision",
		NixOSRelease:      "26.05",
	})
	if err != nil {
		t.Fatal(err)
	}

	extra := filepath.Join(
		destination,
		"oddc",
		"catalog",
		"entities",
		"models",
		"test",
		"unexpected.json",
	)
	if err := os.WriteFile(
		extra,
		[]byte("{}\n"),
		0600,
	); err != nil {
		t.Fatal(err)
	}

	if _, err := Verify(destination); err == nil ||
		!strings.Contains(err.Error(), "unlisted ODDC file") {
		t.Fatalf("verification error=%v", err)
	}
}

func TestMaterializeReplacesExistingCapsuleAndRemovesBackup(t *testing.T) {
	source, identity, resolved := testResolved(t)

	destination := filepath.Join(t.TempDir(), "device-profile")

	first, err := Materialize(MaterializeInput{
		Destination:       destination,
		Identity:          identity,
		Resolved:          resolved,
		Source:            source,
		GjallarOSRevision: "gjallar-old",
		NixOSRelease:      "26.05",
		GenerationReason:  "initial",
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.Source.GjallarOSRevision != "gjallar-old" {
		t.Fatalf("first revision=%q", first.Source.GjallarOSRevision)
	}

	second, err := Materialize(MaterializeInput{
		Destination:       destination,
		Identity:          identity,
		Resolved:          resolved,
		Source:            source,
		GjallarOSRevision: "gjallar-new",
		NixOSRelease:      "26.05",
		GenerationReason:  "hardware-rebind",
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.Source.GjallarOSRevision != "gjallar-new" {
		t.Fatalf("second revision=%q", second.Source.GjallarOSRevision)
	}

	verified, err := Verify(destination)
	if err != nil {
		t.Fatal(err)
	}
	if verified.Source.GjallarOSRevision != "gjallar-new" {
		t.Fatalf(
			"active capsule revision=%q want=%q",
			verified.Source.GjallarOSRevision,
			"gjallar-new",
		)
	}

	if _, err := os.Stat(destination + ".previous"); !os.IsNotExist(err) {
		t.Fatalf("capsule backup still exists after successful activation: %v", err)
	}
}

func TestMaterializeRejectsMissingODDCRevision(t *testing.T) {
	source, identity, resolved := testResolved(t)

	source.Revision = ""
	resolved.Source.Revision = ""

	_, err := Materialize(MaterializeInput{
		Destination:       filepath.Join(t.TempDir(), "device-profile"),
		Identity:          identity,
		Resolved:          resolved,
		Source:            source,
		GjallarOSRevision: "gjallar-test-revision",
		NixOSRelease:      "26.05",
	})
	if err == nil || !strings.Contains(err.Error(), "oddc source revision is required") {
		t.Fatalf("materialize error=%v", err)
	}
}

func TestVerifyRejectsListedCanonicalFileSymlink(t *testing.T) {
	source, identity, resolved := testResolved(t)

	destination := filepath.Join(t.TempDir(), "device-profile")
	capsule, err := Materialize(MaterializeInput{
		Destination:       destination,
		Identity:          identity,
		Resolved:          resolved,
		Source:            source,
		GjallarOSRevision: "gjallar-test-revision",
		NixOSRelease:      "26.05",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(capsule.Manifest.Files) == 0 {
		t.Fatal("test capsule contains no canonical files")
	}

	target := filepath.Join(
		destination,
		filepath.FromSlash(capsule.Manifest.Files[0].Path),
	)

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}

	external := filepath.Join(t.TempDir(), "external-module.nix")
	if err := os.WriteFile(external, data, 0600); err != nil {
		t.Fatal(err)
	}

	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, target); err != nil {
		t.Fatal(err)
	}

	if _, err := Verify(destination); err == nil {
		t.Fatal("listed canonical file symlink was accepted")
	}
}

func TestMaterializeSupportsMachineWithoutODDCProfile(t *testing.T) {
	source := oddc.EmbeddedSource{
		Root:       t.TempDir(),
		Repository: "embedded:oddc",
		Revision:   "oddc-test-revision",
	}
	identity := oddc.Identity{
		FormFactor:  "desktop",
		SysVendor:   "Example Vendor",
		ProductName: "Example Desktop",
		BoardVendor: "Example Vendor",
		BoardName:   "Example Board",
	}

	destination := filepath.Join(t.TempDir(), "device-profile")
	capsule, err := Materialize(MaterializeInput{
		Destination:       destination,
		Identity:          identity,
		Resolved:          oddc.Resolved{},
		Source:            source,
		GjallarOSRevision: "gjallar-test-revision",
		NixOSRelease:      "26.05",
	})
	if err != nil {
		t.Fatal(err)
	}

	if capsule.Manifest.ModelID != "" {
		t.Fatalf(
			"ModelID=%q, want empty for unmatched machine",
			capsule.Manifest.ModelID,
		)
	}
	if capsule.Identity.ProductName != identity.ProductName {
		t.Fatalf("identity not preserved: %+v", capsule.Identity)
	}
	verified, err := Verify(destination)
	if err != nil {
		t.Fatal(err)
	}
	if verified.Manifest.ModelID != "" {
		t.Fatalf(
			"verified ModelID=%q, want empty",
			verified.Manifest.ModelID,
		)
	}
	if _, err := os.Stat(
		filepath.Join(
			destination,
			"oddc",
			"resolved.json",
		),
	); !os.IsNotExist(err) {
		t.Fatalf(
			"unmatched machine unexpectedly has resolved.json: %v",
			err,
		)
	}
}

func TestVerifyRejectsMetadataSymlink(t *testing.T) {
	dir := materializedTestCapsule(t)

	identity := filepath.Join(dir, "identity.json")
	target := filepath.Join(dir, "identity-target.json")

	data, err := os.ReadFile(identity)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(identity); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, identity); err != nil {
		t.Fatal(err)
	}

	_, err = Verify(dir)
	if err == nil || !strings.Contains(err.Error(), "metadata is a symlink") {
		t.Fatalf("expected metadata symlink rejection, got %v", err)
	}
}

func TestVerifyRejectsNonCanonicalManifestPath(t *testing.T) {
	dir := materializedTestCapsule(t)

	manifestPath := filepath.Join(dir, "manifest.json")
	var manifest Manifest

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Files) == 0 {
		t.Fatal("test capsule contains no module files")
	}

	manifest.Files[0].Path = "modules/../modules/" +
		filepath.Base(manifest.Files[0].Path)

	data, err = json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(manifestPath, data, 0600); err != nil {
		t.Fatal(err)
	}

	_, err = Verify(dir)
	if err == nil || !strings.Contains(err.Error(), "non-canonical file path") {
		t.Fatalf("expected non-canonical manifest path rejection, got %v", err)
	}
}

func TestVerifyRejectsMissingResolvedSnapshot(t *testing.T) {
	dir := materializedTestCapsule(t)

	manifestPath := filepath.Join(
		dir,
		"manifest.json",
	)

	var manifest Manifest

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}

	if err := json.Unmarshal(
		data,
		&manifest,
	); err != nil {
		t.Fatal(err)
	}

	files := manifest.Files[:0]
	for _, file := range manifest.Files {
		if file.Path != "oddc/resolved.json" {
			files = append(files, file)
		}
	}
	manifest.Files = files

	data, err = json.MarshalIndent(
		manifest,
		"",
		"  ",
	)
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(
		manifestPath,
		data,
		0600,
	); err != nil {
		t.Fatal(err)
	}

	_, err = Verify(dir)
	if err == nil ||
		!strings.Contains(
			err.Error(),
			"missing resolved.json",
		) {
		t.Fatalf(
			"expected missing resolved snapshot rejection, got %v",
			err,
		)
	}
}
