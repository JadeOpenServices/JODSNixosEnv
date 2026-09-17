package readmodel

import (
	"context"
	"fmt"
	"strings"

	"github.com/bakanura/gjallarOS/pkg/oddc"
)

type modelResolver interface {
	ResolveModel(
		modelID string,
		project []oddc.Overlay,
		host []oddc.Overlay,
	) (oddc.Resolved, error)
}

type registryLoader func(
	root string,
) (modelResolver, error)

// CatalogResolvedSource is the temporary concrete ODDC source used while
// GjallarOS still resolves the catalog directly.
//
// It deliberately receives the selected model ID from its caller rather
// than rediscovering hardware. The future machine-local resolved capsule
// can replace this source without changing Reader or broker contracts.
type CatalogResolvedSource struct {
	Root    string
	ModelID string
	Load    registryLoader
}

func NewCatalogResolvedSource(
	root string,
	modelID string,
) CatalogResolvedSource {
	return CatalogResolvedSource{
		Root:    root,
		ModelID: modelID,
		Load: func(
			path string,
		) (modelResolver, error) {
			return oddc.LoadRegistry(path)
		},
	}
}

func (s CatalogResolvedSource) Resolved(
	ctx context.Context,
) (map[string]any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	root := strings.TrimSpace(s.Root)
	if root == "" {
		return nil, fmt.Errorf(
			"ODDC catalog root is not configured",
		)
	}

	modelID := strings.TrimSpace(s.ModelID)
	if modelID == "" {
		return nil, fmt.Errorf(
			"ODDC model ID is not configured",
		)
	}

	if s.Load == nil {
		return nil, fmt.Errorf(
			"ODDC registry loader is unavailable",
		)
	}

	registry, err := s.Load(root)
	if err != nil {
		return nil, fmt.Errorf(
			"load ODDC registry: %w",
			err,
		)
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	resolved, err := registry.ResolveModel(
		modelID,
		nil,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"resolve ODDC model %q: %w",
			modelID,
			err,
		)
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if resolved.Resolved == nil {
		return nil, fmt.Errorf(
			"ODDC model %q resolved to no data",
			modelID,
		)
	}

	return resolved.Resolved, nil
}
