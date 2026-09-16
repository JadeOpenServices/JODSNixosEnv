package oddcvalidation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bakanura/gjallarOS/internal/installer/oddc"
)

func TestWriteValidationEvidenceUsesCanonicalModel(t *testing.T) {
	repo := t.TempDir()

	validation := oddc.Validation{
		LastValidatedNixOS: "26.05",

		LastValidatedGjallarOSRevision: "git:gjallar",

		LastValidatedDeviceID: "model/framework/laptop-13-amd-ryzen-7040",

		LastValidatedODDCRevision: "git:oddc",

		LastValidatedAt: "2026-09-16T01:55:00Z",
	}

	if err := WriteValidationEvidence(
		repo,
		validation.LastValidatedDeviceID,
		validation,
	); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(
		repo,
		"oddc",
		"evidence",
		"validation",
		"model",
		"framework",
		"laptop-13-amd-ryzen-7040",
		"20260916T015500Z.json",
	)

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var got validationEvidence
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}

	if got.DeviceID !=
		validation.LastValidatedDeviceID {
		t.Fatalf(
			"DeviceID=%q",
			got.DeviceID,
		)
	}

	if got.Status != "hardware-validated" {
		t.Fatalf("Status=%q", got.Status)
	}

	if got.Environment.ODDCRevision !=
		validation.LastValidatedODDCRevision {
		t.Fatalf(
			"ODDCRevision=%q",
			got.Environment.ODDCRevision,
		)
	}

	if _, err := time.Parse(
		time.RFC3339,
		got.Environment.ValidatedAt,
	); err != nil {
		t.Fatal(err)
	}
}
