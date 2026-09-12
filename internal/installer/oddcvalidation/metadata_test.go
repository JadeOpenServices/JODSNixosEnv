package oddcvalidation

import (
	"testing"
	"time"

	"github.com/bakanura/gjallarOS/internal/installer/oddc"
)

func completeReport() Report {
	results := make([]Result, 0, len(RequiredGates))
	for _, gate := range RequiredGates {
		results = append(results, Result{
			Gate:   gate,
			Passed: true,
		})
	}

	return Report{Results: results}
}

func TestValidationMetadataFromCompleteReport(t *testing.T) {
	report := completeReport()

	device := DeviceContext{
		Revision: "git:gjallar123",
		Resolved: oddc.Resolved{
			Device: oddc.Manifest{
				ID: "laptop/framework/13-amd-7040",
			},
			Source: oddc.SourceMetadata{
				Revision: "git:oddc456",
			},
		},
	}

	now := time.Date(
		2026, 9, 11,
		20, 30, 0, 0,
		time.UTC,
	)

	got, err := ValidationMetadata(
		report,
		device,
		"26.05",
		now,
	)
	if err != nil {
		t.Fatal(err)
	}

	if got.LastValidatedNixOS != "26.05" {
		t.Fatalf("LastValidatedNixOS=%q", got.LastValidatedNixOS)
	}
	if got.LastValidatedGjallarOSRevision != "git:gjallar123" {
		t.Fatalf(
			"LastValidatedGjallarOSRevision=%q",
			got.LastValidatedGjallarOSRevision,
		)
	}
	if got.LastValidatedDeviceID != "laptop/framework/13-amd-7040" {
		t.Fatalf("LastValidatedDeviceID=%q", got.LastValidatedDeviceID)
	}
	if got.LastValidatedODDCRevision != "git:oddc456" {
		t.Fatalf(
			"LastValidatedODDCRevision=%q",
			got.LastValidatedODDCRevision,
		)
	}
	if got.LastValidatedAt != "2026-09-11T20:30:00Z" {
		t.Fatalf("LastValidatedAt=%q", got.LastValidatedAt)
	}
}

func TestValidationMetadataRejectsIncompleteReport(t *testing.T) {
	report := Report{
		Results: []Result{
			{
				Gate:   GateGoTests,
				Passed: true,
			},
		},
	}

	_, err := ValidationMetadata(
		report,
		DeviceContext{},
		"26.05",
		time.Now(),
	)

	if err == nil {
		t.Fatal("incomplete validation report produced metadata")
	}
}
