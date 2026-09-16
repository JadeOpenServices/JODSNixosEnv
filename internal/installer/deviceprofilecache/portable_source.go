package deviceprofilecache

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func cachePortableODDCSource(
	sourceRoot string,
	destinationRoot string,
) ([]string, error) {
	roots := []string{
		"catalog/entities",
		"evidence",
	}

	schemaFiles := []string{
		"schemas/entity.schema.json",
		"schemas/evidence.schema.json",
		"schemas/overlay.schema.json",
	}

	var copied []string

	for _, relativeRoot := range roots {
		source := filepath.Join(
			sourceRoot,
			filepath.FromSlash(relativeRoot),
		)

		if _, err := os.Stat(source); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf(
				"inspect canonical ODDC source %s: %w",
				relativeRoot,
				err,
			)
		}

		err := filepath.WalkDir(
			source,
			func(
				path string,
				entry fs.DirEntry,
				walkErr error,
			) error {
				if walkErr != nil {
					return walkErr
				}
				if entry.IsDir() {
					return nil
				}
				if filepath.Ext(path) != ".json" {
					return nil
				}

				rel, err := filepath.Rel(
					sourceRoot,
					path,
				)
				if err != nil {
					return err
				}

				target := filepath.Join(
					destinationRoot,
					rel,
				)

				if err := copyRegularFile(
					path,
					target,
				); err != nil {
					return err
				}

				copied = append(
					copied,
					target,
				)
				return nil
			},
		)
		if err != nil {
			return nil, fmt.Errorf(
				"cache canonical ODDC %s: %w",
				relativeRoot,
				err,
			)
		}
	}

	for _, relative := range schemaFiles {
		source := filepath.Join(
			sourceRoot,
			filepath.FromSlash(relative),
		)

		if _, err := os.Stat(source); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}

		target := filepath.Join(
			destinationRoot,
			filepath.FromSlash(relative),
		)

		if err := copyRegularFile(
			source,
			target,
		); err != nil {
			return nil, err
		}

		copied = append(copied, target)
	}

	return copied, nil
}

func allowedODDCCapsulePath(path string) bool {
	canonical := filepath.ToSlash(
		filepath.Clean(
			filepath.FromSlash(path),
		),
	)

	if canonical == "oddc/resolved.json" {
		return true
	}

	for _, prefix := range []string{
		"oddc/catalog/entities/",
		"oddc/evidence/",
		"oddc/schemas/",
	} {
		if strings.HasPrefix(
			canonical,
			prefix,
		) {
			return true
		}
	}

	return false
}
