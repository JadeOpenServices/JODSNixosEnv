// Package apps reads the app catalogue: every apps/<id>/meta.json.
package apps

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"slices"
)

//go:embed */meta.json
var metadata embed.FS

// App is one catalogue entry; its ID is the folder name.
type App struct {
	ID          string `json:"-"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	// Default selects the app when nobody chose otherwise.
	Default bool `json:"default"`
	// Installer is the installer's yes/no question; empty means the
	// installer never asks and takes Default.
	Installer string `json:"installer"`
}

// Catalogue returns every app, sorted by ID.
func Catalogue() ([]App, error) {
	files, err := fs.Glob(metadata, "*/meta.json")
	if err != nil {
		return nil, err
	}
	slices.Sort(files)
	out := make([]App, 0, len(files))
	for _, file := range files {
		data, err := metadata.ReadFile(file)
		if err != nil {
			return nil, err
		}
		var app App
		if err := json.Unmarshal(data, &app); err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		app.ID = path.Dir(file)
		out = append(out, app)
	}
	return out, nil
}

// Defaults returns the IDs of the apps selected by default.
func Defaults() ([]string, error) {
	catalogue, err := Catalogue()
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for _, app := range catalogue {
		if app.Default {
			ids = append(ids, app.ID)
		}
	}
	return ids, nil
}

// Validate rejects unknown and repeated app IDs.
func Validate(ids []string) error {
	catalogue, err := Catalogue()
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			return fmt.Errorf("apps: %q listed twice", id)
		}
		seen[id] = true
		if !slices.ContainsFunc(catalogue, func(app App) bool { return app.ID == id }) {
			return fmt.Errorf("apps: unknown app %q", id)
		}
	}
	return nil
}
