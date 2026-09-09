package oddc

import "fmt"

type EffectiveSecureBootFirmwarePolicy struct {
	Policy      SecureBootFirmwarePolicy
	SourceLayer string
}

func ResolveSecureBootFirmwarePolicy(
	resolved Resolved,
) (EffectiveSecureBootFirmwarePolicy, error) {
	var effective *EffectiveSecureBootFirmwarePolicy

	for _, layer := range resolved.Inheritance {
		if layer.SecureBootFirmware == nil {
			continue
		}

		if err := validateSecureBootFirmwarePolicy(*layer.SecureBootFirmware); err != nil {
			return EffectiveSecureBootFirmwarePolicy{}, fmt.Errorf(
				"invalid Secure Boot firmware policy on layer %q: %w",
				layer.ID,
				err,
			)
		}

		effective = &EffectiveSecureBootFirmwarePolicy{
			Policy:      cloneSecureBootFirmwarePolicy(*layer.SecureBootFirmware),
			SourceLayer: layer.ID,
		}
	}

	if effective == nil {
		return EffectiveSecureBootFirmwarePolicy{
			Policy: SecureBootFirmwarePolicy{
				Supported:         false,
				SetupModeStrategy: "unsupported",
				UnsupportedReason: "No Secure Boot firmware policy is defined for the resolved device profile.",
			},
		}, nil
	}

	return *effective, nil
}

func cloneSecureBootFirmwarePolicy(
	policy SecureBootFirmwarePolicy,
) SecureBootFirmwarePolicy {
	policy.RequiredPresent = append([]string(nil), policy.RequiredPresent...)
	policy.RequiredAbsent = append([]string(nil), policy.RequiredAbsent...)
	policy.PreserveFirmwareBuiltin = append(
		[]string(nil),
		policy.PreserveFirmwareBuiltin...,
	)
	policy.Untouched = append([]string(nil), policy.Untouched...)
	policy.Instructions = append([]string(nil), policy.Instructions...)
	return policy
}
