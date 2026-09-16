package oddcvalidation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bakanura/gjallarOS/internal/installer/oddc"
)

type validationEvidence struct {
	Schema        string                        `json:"$schema"`
	SchemaVersion string                        `json:"schemaVersion"`
	ID            string                        `json:"id"`
	DeviceID      string                        `json:"deviceId"`
	ObservedAt    string                        `json:"observedAt"`
	Status        string                        `json:"status"`
	Environment   validationEvidenceEnvironment `json:"environment"`
	Results       map[string]string             `json:"results"`
}

type validationEvidenceEnvironment struct {
	OS                string `json:"os"`
	NixOSRelease      string `json:"nixOSRelease"`
	GjallarOSRevision string `json:"gjallarOSRevision"`
	ODDCRevision      string `json:"oddcRevision"`
	ValidatedAt       string `json:"validatedAt"`
}

func WriteValidationEvidence(
	repo string,
	modelID string,
	validation oddc.Validation,
) error {
	modelID = strings.TrimSpace(modelID)

	if !strings.HasPrefix(modelID, "model/") {
		return fmt.Errorf(
			"canonical validation model id must begin with model/: %q",
			modelID,
		)
	}

	if strings.TrimSpace(
		validation.LastValidatedDeviceID,
	) != modelID {
		return fmt.Errorf(
			"validation device id %q does not match canonical model %q",
			validation.LastValidatedDeviceID,
			modelID,
		)
	}

	when, err := time.Parse(
		time.RFC3339,
		validation.LastValidatedAt,
	)
	if err != nil {
		return fmt.Errorf(
			"parse validation timestamp: %w",
			err,
		)
	}
	when = when.UTC()

	slug := strings.NewReplacer(
		"/", "-",
		"_", "-",
		".", "-",
	).Replace(strings.ToLower(modelID))

	stamp := strings.ToLower(
		when.Format("20060102T150405Z"),
	)

	evidence := validationEvidence{
		Schema: "https://openjade.de/oddc/schemas/evidence.schema.json",

		SchemaVersion: "2.0.0",

		ID: "validation-" + slug + "-" + stamp,

		DeviceID: modelID,

		ObservedAt: when.Format("2006-01-02"),

		Status: "hardware-validated",

		Environment: validationEvidenceEnvironment{
			OS: "GjallarOS",

			NixOSRelease: validation.LastValidatedNixOS,

			GjallarOSRevision: validation.LastValidatedGjallarOSRevision,

			ODDCRevision: validation.LastValidatedODDCRevision,

			ValidatedAt: validation.LastValidatedAt,
		},

		Results: map[string]string{
			"realDeviceValidation": "pass",
		},
	}

	data, err := json.MarshalIndent(
		evidence,
		"",
		"  ",
	)
	if err != nil {
		return fmt.Errorf(
			"encode canonical validation evidence: %w",
			err,
		)
	}
	data = append(data, '\n')

	dir := filepath.Join(
		repo,
		"oddc",
		"evidence",
		"validation",
		filepath.FromSlash(modelID),
	)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf(
			"create validation evidence directory: %w",
			err,
		)
	}

	path := filepath.Join(
		dir,
		when.Format("20060102T150405Z")+".json",
	)

	tmp, err := os.CreateTemp(
		dir,
		".validation-evidence-*",
	)
	if err != nil {
		return err
	}

	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if err := tmp.Chmod(0644); err != nil {
		tmp.Close()
		return err
	}

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}

	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}

	if err := tmp.Close(); err != nil {
		return err
	}

	if err := os.Rename(
		tmpPath,
		path,
	); err != nil {
		return fmt.Errorf(
			"install canonical validation evidence: %w",
			err,
		)
	}

	return nil
}
