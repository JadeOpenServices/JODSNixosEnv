package oddcvalidation

import (
	"testing"

	"github.com/bakanura/gjallarOS/internal/installer/oddc"
)

func TestSecureBootPolicyGateRejectsMissingPolicy(t *testing.T) {
	resolved := oddc.Resolved{
		Device: oddc.Manifest{ID: "laptop/test"},
		Inheritance: []oddc.Manifest{
			{ID: "laptop/common"},
			{ID: "laptop/test"},
		},
	}

	result := secureBootPolicyResult(resolved)
	if result.Passed {
		t.Fatal("missing Secure Boot policy was accepted")
	}
}

func TestSecureBootPolicyGateAcceptsExplicitUnsupportedPolicy(t *testing.T) {
	resolved := oddc.Resolved{
		Device: oddc.Manifest{ID: "laptop/test"},
		Inheritance: []oddc.Manifest{
			{
				ID: "laptop/test",
				SecureBootFirmware: &oddc.SecureBootFirmwarePolicy{
					Supported:         false,
					SetupModeStrategy: "unsupported",
					UnsupportedReason: "firmware workflow has not been validated",
				},
			},
		},
	}

	result := secureBootPolicyResult(resolved)
	if !result.Passed {
		t.Fatalf("explicit unsupported policy rejected: %+v", result)
	}
}
