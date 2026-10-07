package oddchost

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/JadeOpenServices/oddc/pkg/oddc"
)

const (
	overlaySchema = "https://openjade.de/oddc/schemas/overlay.schema.json"
	overlayID     = "host/machine-local"

	RelativePath = "var/lib/gjallarOS/oddc/host-overlay.json"
)

// Path returns the durable machine-local host-overlay path beneath a mounted
// system root. "/" addresses the running system; "/mnt" addresses the
// recovery-mounted installed system.
func Path(root string) (string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return "", fmt.Errorf("ODDC host overlay root is required")
	}
	if !filepath.IsAbs(root) {
		return "", fmt.Errorf(
			"ODDC host overlay root must be absolute: %q",
			root,
		)
	}

	return filepath.Join(root, filepath.FromSlash(RelativePath)), nil
}

// Load reads a machine-local host overlay and verifies that it belongs to the
// requested canonical model. A missing file is reported as exists=false.
func Load(
	path string,
	modelID string,
) (overlay oddc.Overlay, exists bool, err error) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return oddc.Overlay{}, false, fmt.Errorf(
			"ODDC host overlay model id is required",
		)
	}

	overlay, err = oddc.ReadOverlay(path)
	if err != nil {
		if os.IsNotExist(err) {
			return oddc.Overlay{}, false, nil
		}

		return oddc.Overlay{}, false, err
	}

	if err := validateLoadedOverlay(overlay, modelID); err != nil {
		return oddc.Overlay{}, false, err
	}

	return overlay, true, nil
}

func validateLoadedOverlay(
	overlay oddc.Overlay,
	modelID string,
) error {
	if overlay.APIVersion != oddc.EntityAPIVersion {
		return fmt.Errorf(
			"ODDC host overlay apiVersion=%q want=%q",
			overlay.APIVersion,
			oddc.EntityAPIVersion,
		)
	}

	if overlay.Kind != "host" {
		return fmt.Errorf(
			"ODDC machine-local overlay kind=%q want=host",
			overlay.Kind,
		)
	}

	if strings.TrimSpace(overlay.ID) == "" {
		return fmt.Errorf("ODDC host overlay id is required")
	}

	if strings.TrimSpace(overlay.TargetModel) == "" {
		return fmt.Errorf("ODDC host overlay target model is required")
	}

	if overlay.TargetModel != modelID {
		return fmt.Errorf(
			"ODDC host overlay targets %q, current model is %q",
			overlay.TargetModel,
			modelID,
		)
	}

	return nil
}

// New creates empty machine-local ODDC host state for one canonical model.
func New(modelID string) (oddc.Overlay, error) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return oddc.Overlay{}, fmt.Errorf(
			"ODDC host overlay model id is required",
		)
	}

	return oddc.Overlay{
		Schema:      overlaySchema,
		APIVersion:  oddc.EntityAPIVersion,
		ID:          overlayID,
		Kind:        "host",
		TargetModel: modelID,
		Overrides:   map[string]any{},
	}, nil
}

// AddDeletion stages one intentional machine-local hardware removal entirely
// in memory. It does not perform any filesystem operation.
func AddDeletion(
	overlay *oddc.Overlay,
	resolvedPath string,
) error {
	if overlay == nil {
		return fmt.Errorf("ODDC host overlay is required")
	}

	if err := validateLoadedOverlay(
		*overlay,
		overlay.TargetModel,
	); err != nil {
		return err
	}

	parts, err := resolvedPathParts(resolvedPath)
	if err != nil {
		return err
	}

	if overlay.Overrides == nil {
		overlay.Overrides = map[string]any{}
	}

	if err := setDeletion(overlay.Overrides, parts); err != nil {
		return fmt.Errorf(
			"record ODDC host deletion %q: %w",
			resolvedPath,
			err,
		)
	}

	return nil
}

// RecordDeletion records that one resolved ODDC path is intentionally absent
// on this machine. Existing unrelated host overrides are preserved.
func RecordDeletion(
	path string,
	modelID string,
	resolvedPath string,
) error {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return fmt.Errorf("ODDC host overlay model id is required")
	}

	overlay, exists, err := Load(path, modelID)
	if err != nil {
		return err
	}

	if !exists {
		overlay, err = New(modelID)
		if err != nil {
			return err
		}
	}

	if err := AddDeletion(&overlay, resolvedPath); err != nil {
		return err
	}

	return writeAtomic(path, overlay)
}

func resolvedPathParts(path string) ([]string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("ODDC resolved path is required")
	}

	parts := strings.Split(path, ".")
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return nil, fmt.Errorf(
				"invalid ODDC resolved path %q",
				path,
			)
		}

		if part == "$delete" {
			return nil, fmt.Errorf(
				"ODDC resolved path may not contain $delete",
			)
		}
	}

	return parts, nil
}

func setDeletion(
	root map[string]any,
	parts []string,
) error {
	current := root

	for index, part := range parts {
		last := index == len(parts)-1
		if last {
			current[part] = map[string]any{
				"$delete": true,
			}
			return nil
		}

		existing, exists := current[part]
		if !exists {
			next := map[string]any{}
			current[part] = next
			current = next
			continue
		}

		next, ok := existing.(map[string]any)
		if !ok {
			return fmt.Errorf(
				"path component %q already has a scalar override",
				strings.Join(parts[:index+1], "."),
			)
		}

		if deleted, ok := next["$delete"].(bool); ok && deleted {
			return fmt.Errorf(
				"path component %q is already deleted",
				strings.Join(parts[:index+1], "."),
			)
		}

		current = next
	}

	return nil
}

func writeAtomic(
	path string,
	overlay oddc.Overlay,
) error {
	data, err := json.MarshalIndent(overlay, "", "  ")
	if err != nil {
		return fmt.Errorf("encode ODDC host overlay: %w", err)
	}
	data = append(data, '\n')

	dir := filepath.Dir(filepath.Clean(path))
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create ODDC host overlay directory: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".oddc-host-overlay-*")
	if err != nil {
		return fmt.Errorf("create ODDC host overlay temporary file: %w", err)
	}

	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if err := tmp.Chmod(0600); err != nil {
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

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("activate ODDC host overlay: %w", err)
	}

	dirHandle, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("open ODDC host overlay directory: %w", err)
	}
	defer dirHandle.Close()

	if err := dirHandle.Sync(); err != nil {
		return fmt.Errorf("sync ODDC host overlay directory: %w", err)
	}

	return nil
}
