package oddcvalidation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/bakanura/gjallarOS/internal/installer/oddc"
)

func TestUpdateDeviceValidationOnlyChangesValidation(t *testing.T) {
	repo := t.TempDir()

	path := filepath.Join(
		repo,
		"oddc",
		"devices",
		"laptop",
		"test",
		"device.json",
	)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}

	original := oddc.Manifest{
		Schema: 1,
		ID:     "laptop/test",
		Class:  "laptop",
		Lifecycle: oddc.Lifecycle{
			Status: "supported",
		},
	}

	data, err := json.MarshalIndent(original, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	want := validTestValidation()

	if err := UpdateDeviceValidation(
		repo,
		"laptop/test",
		want,
	); err != nil {
		t.Fatal(err)
	}

	updatedData, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var got oddc.Manifest
	if err := json.Unmarshal(updatedData, &got); err != nil {
		t.Fatal(err)
	}

	if got.ID != original.ID {
		t.Fatalf("ID changed: %q", got.ID)
	}
	if got.Class != original.Class {
		t.Fatalf("Class changed: %q", got.Class)
	}
	if got.Lifecycle.Status != original.Lifecycle.Status {
		t.Fatalf(
			"Lifecycle.Status changed: %q",
			got.Lifecycle.Status,
		)
	}

	if got.Validation != want {
		t.Fatalf(
			"Validation=%+v want %+v",
			got.Validation,
			want,
		)
	}
}

func TestUpdateDeviceValidationRejectsWrongManifest(t *testing.T) {
	repo := t.TempDir()

	path := filepath.Join(
		repo,
		"oddc",
		"devices",
		"laptop",
		"test",
		"device.json",
	)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		path,
		[]byte(`{
		  "schema": 1,
		  "id": "laptop/other",
		  "class": "laptop",
		  "lifecycle": {"status": "supported"},
		  "validation": {}
		}`+"\n"),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	err := UpdateDeviceValidation(
		repo,
		"laptop/test",
		validTestValidation(),
	)
	if err == nil {
		t.Fatal("mismatched manifest id was accepted")
	}
}
