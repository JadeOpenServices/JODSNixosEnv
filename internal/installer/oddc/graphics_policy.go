package oddc

import (
	"fmt"
	"strings"

	portable "github.com/JadeOpenServices/oddc"
)

type EffectiveGraphicsPolicy struct {
	Policy       GraphicsPolicy
	SourceEntity string
}

func ResolveGraphicsPolicy(
	resolved Resolved,
) (EffectiveGraphicsPolicy, error) {
	if resolved.ModelID == "" ||
		resolved.Canonical.Resolved == nil {
		return EffectiveGraphicsPolicy{}, nil
	}

	policy := GraphicsPolicy{}

	if value, ok := portable.Lookup(
		resolved.Canonical.Resolved,
		"hardware.graphics.integrated.driver",
	); ok {
		policy.IntegratedKernelDriver, _ = value.(string)
	}

	if value, ok := portable.Lookup(
		resolved.Canonical.Resolved,
		"hardware.graphics.discrete.driver",
	); ok {
		policy.DiscreteKernelDriver, _ = value.(string)
	}

	if value, ok := portable.Lookup(
		resolved.Canonical.Resolved,
		"policy.graphics.discrete.driverBranch",
	); ok {
		policy.DriverBranch, _ = value.(string)
	}

	if policy.IntegratedKernelDriver == "" &&
		policy.DiscreteKernelDriver == "" &&
		policy.DriverBranch == "" {
		return EffectiveGraphicsPolicy{}, nil
	}

	if err := validateGraphicsPolicy(policy); err != nil {
		return EffectiveGraphicsPolicy{}, err
	}

	sourcePath := "policy.graphics.discrete.driverBranch"
	if policy.DriverBranch == "" {
		sourcePath = "hardware.graphics.discrete.driver"
	}
	if policy.DiscreteKernelDriver == "" {
		sourcePath = "hardware.graphics.integrated.driver"
	}

	source := ""
	if provenance, ok :=
		resolved.Canonical.Provenance[sourcePath]; ok {
		source = strings.TrimPrefix(
			provenance.Source,
			"catalog:",
		)
	}

	return EffectiveGraphicsPolicy{
		Policy:       policy,
		SourceEntity: source,
	}, nil
}

func validateGraphicsPolicy(
	policy GraphicsPolicy,
) error {
	branch := strings.TrimSpace(policy.DriverBranch)

	if strings.TrimSpace(policy.DiscreteKernelDriver) == "" &&
		branch == "" {
		return nil
	}

	switch branch {
	case "stable", "legacy_580":
		return nil

	case "":
		return fmt.Errorf(
			"driverBranch must not be empty for discrete graphics",
		)

	default:
		return fmt.Errorf(
			"unsupported driverBranch %q",
			policy.DriverBranch,
		)
	}
}
