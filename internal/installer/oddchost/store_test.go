package oddchost

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bakanura/gjallarOS/pkg/oddc"
)

const testModel = "model/test/laptop"

func TestRecordDeletionCreatesHostOverlay(t *testing.T) {
	path := filepath.Join(t.TempDir(), "host.json")

	if err := RecordDeletion(
		path,
		testModel,
		"hardware.security.fingerprint.primary",
	); err != nil {
		t.Fatal(err)
	}

	overlay, exists, err := Load(path, testModel)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("written host overlay does not exist")
	}

	if overlay.Kind != "host" ||
		overlay.TargetModel != testModel {
		t.Fatalf("unexpected host overlay: %+v", overlay)
	}

	value, ok := lookup(
		overlay.Overrides,
		"hardware.security.fingerprint.primary.$delete",
	)
	if !ok || value != true {
		t.Fatalf("deletion tombstone missing: %#v", overlay.Overrides)
	}
}

func TestRecordDeletionPreservesExistingOverrides(t *testing.T) {
	path := filepath.Join(t.TempDir(), "host.json")

	initial := oddc.Overlay{
		Schema:      overlaySchema,
		APIVersion:  oddc.EntityAPIVersion,
		ID:          overlayID,
		Kind:        "host",
		TargetModel: testModel,
		Overrides: map[string]any{
			"policy": map[string]any{
				"thermal": map[string]any{
					"profile": "quiet",
				},
			},
		},
	}

	if err := writeAtomic(path, initial); err != nil {
		t.Fatal(err)
	}

	for _, resolvedPath := range []string{
		"hardware.security.fingerprint.primary",
		"hardware.radio.bluetooth.primary",
	} {
		if err := RecordDeletion(
			path,
			testModel,
			resolvedPath,
		); err != nil {
			t.Fatal(err)
		}
	}

	overlay, exists, err := Load(path, testModel)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("host overlay disappeared")
	}

	value, ok := lookup(
		overlay.Overrides,
		"policy.thermal.profile",
	)
	if !ok || value != "quiet" {
		t.Fatalf(
			"unrelated host override was lost: %#v",
			overlay.Overrides,
		)
	}

	for _, resolvedPath := range []string{
		"hardware.security.fingerprint.primary.$delete",
		"hardware.radio.bluetooth.primary.$delete",
	} {
		value, ok := lookup(overlay.Overrides, resolvedPath)
		if !ok || value != true {
			t.Fatalf(
				"missing deletion %s: %#v",
				resolvedPath,
				overlay.Overrides,
			)
		}
	}
}

func TestLoadRejectsDifferentModel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "host.json")

	if err := RecordDeletion(
		path,
		testModel,
		"hardware.security.fingerprint.primary",
	); err != nil {
		t.Fatal(err)
	}

	if _, _, err := Load(
		path,
		"model/test/different-laptop",
	); err == nil {
		t.Fatal("host overlay was accepted for a different model")
	}
}

func TestRecordDeletionRejectsInvalidResolvedPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "host.json")

	for _, resolvedPath := range []string{
		"",
		"hardware..primary",
		"hardware.$delete.primary",
	} {
		if err := RecordDeletion(
			path,
			testModel,
			resolvedPath,
		); err == nil {
			t.Fatalf(
				"invalid resolved path %q was accepted",
				resolvedPath,
			)
		}
	}
}

func lookup(
	root map[string]any,
	path string,
) (any, bool) {
	return oddc.Lookup(root, path)
}

func TestPathUsesMountedSystemRoot(t *testing.T) {
	tests := []struct {
		root string
		want string
	}{
		{
			root: "/",
			want: "/var/lib/gjallarOS/oddc/host-overlay.json",
		},
		{
			root: "/mnt",
			want: "/mnt/var/lib/gjallarOS/oddc/host-overlay.json",
		},
	}

	for _, test := range tests {
		got, err := Path(test.root)
		if err != nil {
			t.Fatal(err)
		}
		if got != test.want {
			t.Fatalf(
				"Path(%q) = %q, want %q",
				test.root,
				got,
				test.want,
			)
		}
	}
}

func TestPathRejectsNonAbsoluteRoot(t *testing.T) {
	for _, root := range []string{"", ".", "mnt"} {
		if _, err := Path(root); err == nil {
			t.Fatalf("Path(%q) accepted invalid root", root)
		}
	}
}

func TestRecordDeletionProtectsMachineLocalState(t *testing.T) {
	root := t.TempDir()
	path, err := Path(root)
	if err != nil {
		t.Fatal(err)
	}

	if err := RecordDeletion(
		path,
		testModel,
		"hardware.security.fingerprint.primary",
	); err != nil {
		t.Fatal(err)
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fileInfo.Mode().Perm(); got != 0600 {
		t.Fatalf("host overlay mode = %04o, want 0600", got)
	}

	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if got := dirInfo.Mode().Perm(); got != 0700 {
		t.Fatalf("host overlay directory mode = %04o, want 0700", got)
	}
}

func TestNewCreatesModelBoundHostOverlay(t *testing.T) {
	overlay, err := New(testModel)
	if err != nil {
		t.Fatal(err)
	}

	if overlay.Kind != "host" {
		t.Fatalf("kind = %q", overlay.Kind)
	}
	if overlay.TargetModel != testModel {
		t.Fatalf(
			"target model = %q, want %q",
			overlay.TargetModel,
			testModel,
		)
	}
	if overlay.Overrides == nil {
		t.Fatal("new host overlay has nil overrides")
	}
}

func TestAddDeletionMutatesOnlyMemory(t *testing.T) {
	overlay, err := New(testModel)
	if err != nil {
		t.Fatal(err)
	}

	overlay.Overrides["policy"] = map[string]any{
		"thermal": map[string]any{
			"profile": "quiet",
		},
	}

	if err := AddDeletion(
		&overlay,
		"hardware.security.fingerprint.primary",
	); err != nil {
		t.Fatal(err)
	}

	deleted, ok := lookup(
		overlay.Overrides,
		"hardware.security.fingerprint.primary.$delete",
	)
	if !ok || deleted != true {
		t.Fatalf(
			"in-memory deletion missing: %#v",
			overlay.Overrides,
		)
	}

	profile, ok := lookup(
		overlay.Overrides,
		"policy.thermal.profile",
	)
	if !ok || profile != "quiet" {
		t.Fatalf(
			"unrelated override changed: %#v",
			overlay.Overrides,
		)
	}
}

func TestAddDeletionRejectsInvalidOverlay(t *testing.T) {
	overlay := oddc.Overlay{
		APIVersion:  oddc.EntityAPIVersion,
		ID:          "project/not-host",
		Kind:        "project",
		TargetModel: testModel,
		Overrides:   map[string]any{},
	}

	if err := AddDeletion(
		&overlay,
		"hardware.security.fingerprint.primary",
	); err == nil {
		t.Fatal("non-host overlay accepted for machine-local deletion")
	}
}

func TestNewRejectsEmptyModel(t *testing.T) {
	if _, err := New(" "); err == nil {
		t.Fatal("empty host-overlay model accepted")
	}
}
