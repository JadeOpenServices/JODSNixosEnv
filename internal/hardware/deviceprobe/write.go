package deviceprobe

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func WriteSnapshot(path string, snapshot Snapshot) error {
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("encode device probe snapshot: %w", err)
	}
	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create device probe directory: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".device-probe-*")
	if err != nil {
		return fmt.Errorf("create temporary device probe snapshot: %w", err)
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
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace device probe snapshot: %w", err)
	}

	return nil
}

func ReadSnapshot(path string) (Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Snapshot{}, fmt.Errorf("read device probe snapshot: %w", err)
	}

	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return Snapshot{}, fmt.Errorf("decode device probe snapshot: %w", err)
	}

	if snapshot.Schema != 1 {
		return Snapshot{}, fmt.Errorf(
			"unsupported device probe schema %d",
			snapshot.Schema,
		)
	}

	return snapshot, nil
}
