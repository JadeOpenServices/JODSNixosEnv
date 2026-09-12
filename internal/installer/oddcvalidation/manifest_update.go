package oddcvalidation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bakanura/gjallarOS/internal/installer/oddc"
)

func UpdateDeviceValidation(
	repo string,
	deviceID string,
	validation oddc.Validation,
) error {
	if deviceID == "" {
		return fmt.Errorf("device id is empty")
	}

	path := filepath.Join(
		repo,
		"oddc",
		"devices",
		filepath.FromSlash(deviceID),
		"device.json",
	)

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read ODDC device manifest: %w", err)
	}

	var manifest oddc.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("decode ODDC device manifest: %w", err)
	}

	if manifest.ID != deviceID {
		return fmt.Errorf(
			"ODDC manifest id %q does not match expected device %q",
			manifest.ID,
			deviceID,
		)
	}

	manifest.Validation = validation

	rendered, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode ODDC device manifest: %w", err)
	}
	rendered = append(rendered, '\n')

	tmp, err := os.CreateTemp(
		filepath.Dir(path),
		".device.json-*",
	)
	if err != nil {
		return fmt.Errorf("create temporary ODDC manifest: %w", err)
	}

	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if err := tmp.Chmod(0644); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(rendered); err != nil {
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

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace ODDC device manifest: %w", err)
	}

	return nil
}
