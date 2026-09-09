package oddc

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrNoMatch        = errors.New("no oddc device profile matched")
	ErrAmbiguousMatch = errors.New("ambiguous oddc device profile match")
)

type SourceMetadata struct {
	Kind       string
	Repository string
	Revision   string
	Integrity  string
}

type Resolved struct {
	Device      Manifest
	Inheritance []Manifest
	Source      SourceMetadata

	directories map[string]string
}

type Materialized struct {
	Files []string
}

type DeviceSource interface {
	Resolve(identity Identity) (Resolved, error)
	Materialize(resolved Resolved, destination string) (Materialized, error)
	Metadata() SourceMetadata
}

type EmbeddedSource struct {
	Root       string
	Repository string
	Revision   string
	Integrity  string
}

func (source EmbeddedSource) Metadata() SourceMetadata {
	return SourceMetadata{
		Kind:       "embedded",
		Repository: source.Repository,
		Revision:   source.Revision,
		Integrity:  source.Integrity,
	}
}

type manifestRecord struct {
	manifest Manifest
	dir      string
}

func (source EmbeddedSource) Resolve(identity Identity) (Resolved, error) {
	records, err := source.load()
	if err != nil {
		return Resolved{}, err
	}

	bestScore := -1
	var best []manifestRecord

	for _, record := range records {
		matched, score := matchIdentity(record.manifest.Match, identity)
		if !matched {
			continue
		}

		switch {
		case score > bestScore:
			bestScore = score
			best = []manifestRecord{record}
		case score == bestScore:
			best = append(best, record)
		}
	}

	if len(best) == 0 {
		commonID := strings.TrimSpace(strings.ToLower(identity.FormFactor)) + "/common"
		if commonID == "/common" {
			return Resolved{}, ErrNoMatch
		}

		for _, record := range records {
			if record.manifest.ID == commonID {
				best = []manifestRecord{record}
				bestScore = 0
				break
			}
		}

		if len(best) == 0 {
			return Resolved{}, ErrNoMatch
		}
	}

	if len(best) != 1 {
		ids := make([]string, 0, len(best))
		for _, record := range best {
			ids = append(ids, record.manifest.ID)
		}
		return Resolved{}, fmt.Errorf(
			"%w: %s",
			ErrAmbiguousMatch,
			strings.Join(ids, ", "),
		)
	}

	index := make(map[string]manifestRecord, len(records))
	for _, record := range records {
		index[record.manifest.ID] = record
	}

	ordered, err := resolveInheritance(best[0].manifest.ID, index)
	if err != nil {
		return Resolved{}, err
	}

	directories := make(map[string]string, len(ordered))
	manifests := make([]Manifest, 0, len(ordered))
	for _, record := range ordered {
		manifests = append(manifests, record.manifest)
		directories[record.manifest.ID] = record.dir
	}

	return Resolved{
		Device:      best[0].manifest,
		Inheritance: manifests,
		Source:      source.Metadata(),
		directories: directories,
	}, nil
}

func (source EmbeddedSource) load() ([]manifestRecord, error) {
	root := filepath.Join(source.Root, "devices")

	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("inspect embedded oddc source: %w", err)
	}

	var records []manifestRecord
	seen := map[string]string{}

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || entry.Name() != "device.json" {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("open oddc manifest %s: %w", path, err)
		}
		defer file.Close()

		manifest, err := DecodeManifest(file)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}

		relativeDir, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return fmt.Errorf("resolve oddc manifest directory %s: %w", path, err)
		}
		expectedID := filepath.ToSlash(relativeDir)
		if manifest.ID != expectedID {
			return fmt.Errorf(
				"oddc manifest id %q does not match directory %q",
				manifest.ID,
				expectedID,
			)
		}

		if previous, exists := seen[manifest.ID]; exists {
			return fmt.Errorf(
				"duplicate oddc device id %q in %s and %s",
				manifest.ID,
				previous,
				path,
			)
		}

		seen[manifest.ID] = path
		records = append(records, manifestRecord{
			manifest: manifest,
			dir:      filepath.Dir(path),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("load embedded oddc source: %w", err)
	}

	return records, nil
}

func resolveInheritance(
	id string,
	index map[string]manifestRecord,
) ([]manifestRecord, error) {
	visited := map[string]bool{}
	active := map[string]bool{}
	var ordered []manifestRecord

	var visit func(string) error
	visit = func(current string) error {
		if active[current] {
			return fmt.Errorf("oddc inheritance cycle at %q", current)
		}
		if visited[current] {
			return nil
		}

		record, ok := index[current]
		if !ok {
			return fmt.Errorf("oddc inherited device profile %q does not exist", current)
		}

		active[current] = true
		for _, parent := range record.manifest.Inherits {
			if err := visit(parent); err != nil {
				return err
			}
		}
		delete(active, current)

		visited[current] = true
		ordered = append(ordered, record)
		return nil
	}

	if err := visit(id); err != nil {
		return nil, err
	}

	return ordered, nil
}

func matchIdentity(match Match, identity Identity) (bool, int) {
	checks := []struct {
		values []string
		actual string
	}{
		{match.SysVendor, identity.SysVendor},
		{match.ProductName, identity.ProductName},
		{match.ProductVersion, identity.ProductVersion},
		{match.BoardVendor, identity.BoardVendor},
		{match.BoardName, identity.BoardName},
		{match.BoardVersion, identity.BoardVersion},
	}

	score := 0
	for _, check := range checks {
		if len(check.values) == 0 {
			continue
		}

		found := false
		for _, expected := range check.values {
			if strings.EqualFold(
				strings.TrimSpace(expected),
				strings.TrimSpace(check.actual),
			) {
				found = true
				break
			}
		}
		if !found {
			return false, 0
		}
		score++
	}

	// Matchless manifests are inheritance layers, not concrete device matches.
	return score > 0, score
}

func (source EmbeddedSource) Materialize(
	resolved Resolved,
	destination string,
) (Materialized, error) {
	var result Materialized

	for _, manifest := range resolved.Inheritance {
		dir, ok := resolved.directories[manifest.ID]
		if !ok {
			return Materialized{}, fmt.Errorf(
				"missing source directory for oddc profile %q",
				manifest.ID,
			)
		}

		for _, module := range manifest.Modules {
			if err := validateRelativePath(module); err != nil {
				return Materialized{}, err
			}

			sourcePath, err := confinedModulePath(dir, module)
			if err != nil {
				return Materialized{}, fmt.Errorf(
					"oddc manifest %q module %q: %w",
					manifest.ID,
					module,
					err,
				)
			}

			targetPath := filepath.Join(
				destination,
				filepath.FromSlash(manifest.ID),
				module,
			)

			if err := copyRegularFile(sourcePath, targetPath); err != nil {
				return Materialized{}, err
			}

			result.Files = append(result.Files, targetPath)
		}
	}

	return result, nil
}

func confinedModulePath(deviceDir, module string) (string, error) {
	deviceRoot, err := filepath.EvalSymlinks(deviceDir)
	if err != nil {
		return "", fmt.Errorf("resolve oddc device directory: %w", err)
	}

	candidate := filepath.Join(deviceDir, module)
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", fmt.Errorf("resolve oddc module: %w", err)
	}

	rel, err := filepath.Rel(deviceRoot, resolved)
	if err != nil {
		return "", fmt.Errorf("compare oddc module path: %w", err)
	}
	if rel == ".." ||
		strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("resolved module escapes device directory")
	}

	return resolved, nil
}

func copyRegularFile(source, target string) error {
	in, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open oddc module %s: %w", source, err)
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return fmt.Errorf("inspect oddc module %s: %w", source, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("oddc module %s is not a regular file", source)
	}

	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return fmt.Errorf("create oddc materialization directory: %w", err)
	}

	out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("create oddc materialized module %s: %w", target, err)
	}

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return fmt.Errorf("copy oddc module %s: %w", source, err)
	}

	if err := out.Sync(); err != nil {
		out.Close()
		return fmt.Errorf("sync oddc module %s: %w", target, err)
	}

	return out.Close()
}
