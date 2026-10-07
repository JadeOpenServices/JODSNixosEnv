package oddc

import (
	"errors"
	"fmt"
	"os"

	portable "github.com/JadeOpenServices/oddc/pkg/oddc"
)

// AnswerRevision is the ODDC commit the answer in root came from, or ""
// without an answer.
func AnswerRevision(root string) string {
	if _, err := os.Stat(root); errors.Is(err, os.ErrNotExist) {
		return ""
	}

	return portable.DirSource{Root: root}.Revision()
}

// AnswerModel is the one DeviceModel the answer in root holds.
func AnswerModel(root string) (string, error) {
	registry, err := portable.LoadRegistry(root)
	if err != nil {
		return "", fmt.Errorf("read ODDC answer: %w", err)
	}

	var models []string
	for id, entity := range registry.Entities {
		if entity.Kind == "DeviceModel" {
			models = append(models, id)
		}
	}
	if len(models) != 1 {
		return "", fmt.Errorf("ODDC answer in %s holds %d models, want 1", root, len(models))
	}

	return models[0], nil
}
