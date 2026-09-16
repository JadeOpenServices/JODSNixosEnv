package oddc

import (
	"fmt"
	"strings"
)

type EffectiveGraphicsPolicy struct {
	Policy      GraphicsPolicy
	SourceLayer string
}

func ResolveGraphicsPolicy(resolved Resolved) (EffectiveGraphicsPolicy, error) {
	var effective *EffectiveGraphicsPolicy

	for _, layer := range resolved.Inheritance {
		if layer.Graphics == nil {
			continue
		}

		if err := validateGraphicsPolicy(*layer.Graphics); err != nil {
			return EffectiveGraphicsPolicy{}, fmt.Errorf(
				"invalid graphics policy on layer %q: %w",
				layer.ID,
				err,
			)
		}

		effective = &EffectiveGraphicsPolicy{
			Policy:      *layer.Graphics,
			SourceLayer: layer.ID,
		}
	}

	if effective == nil {
		return EffectiveGraphicsPolicy{}, nil
	}

	return *effective, nil
}

func validateGraphicsPolicy(policy GraphicsPolicy) error {
	switch strings.TrimSpace(policy.DriverBranch) {
	case "stable", "legacy_580":
		return nil
	case "":
		return fmt.Errorf("driverBranch must not be empty")
	default:
		return fmt.Errorf("unsupported driverBranch %q", policy.DriverBranch)
	}
}
