package readmodel

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

// ResolvedFile consumes the same fully resolved view used by NixOS, including
// host overrides. It never rediscovers the machine or re-resolves the catalog.
type ResolvedFile string

func (path ResolvedFile) Resolved(ctx context.Context) (map[string]any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(string(path))
	if err != nil {
		return nil, err
	}
	var resolved map[string]any
	if err := json.Unmarshal(data, &resolved); err != nil {
		return nil, err
	}
	if resolved == nil {
		return nil, fmt.Errorf("resolved ODDC view is empty")
	}
	return resolved, nil
}
