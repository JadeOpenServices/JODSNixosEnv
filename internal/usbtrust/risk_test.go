package usbtrust

import (
	"testing"
)

func TestRiskDetectsHIDInjectionCapability(t *testing.T) {
	findings := mustRisk(t, Identity{
		Interfaces: []string{"03:01:01"},
	})

	requireRisk(t, findings, RiskHIDInput, RiskHigh)
}

func TestRiskDetectsMassStorage(t *testing.T) {
	findings := mustRisk(t, Identity{
		Interfaces: []string{"08:06:50"},
	})

	requireRisk(t, findings, RiskMassStorage, RiskHigh)
}

func TestRiskDetectsNetworkInterfaces(t *testing.T) {
	findings := mustRisk(t, Identity{
		Interfaces: []string{
			"02:0d:00",
			"0a:00:00",
		},
	})

	requireRisk(t, findings, RiskNetwork, RiskHigh)
}

func TestRiskDetectsHIDStorageComposite(t *testing.T) {
	findings := mustRisk(t, Identity{
		Interfaces: []string{
			"03:01:01",
			"08:06:50",
		},
	})

	requireRisk(t, findings, RiskHIDInput, RiskHigh)
	requireRisk(t, findings, RiskMassStorage, RiskHigh)
	requireRisk(t, findings, RiskComposite, RiskReview)
	requireRisk(
		t,
		findings,
		RiskHIDStorageComposite,
		RiskHigh,
	)
}

func TestRiskDetectsOpaqueVendorInterface(t *testing.T) {
	findings := mustRisk(t, Identity{
		Interfaces: []string{"ff:00:00"},
	})

	requireRisk(t, findings, RiskVendorSpecific, RiskReview)
}

func TestRepeatedInterfaceDoesNotCreateCompositeFinding(
	t *testing.T,
) {
	findings := mustRisk(t, Identity{
		Interfaces: []string{
			"e0:01:01",
			"e0:01:01",
			"e0:01:01",
		},
	})

	requireRisk(
		t,
		findings,
		RiskWirelessController,
		RiskReview,
	)

	if hasRisk(findings, RiskComposite) {
		t.Fatal("duplicate interface descriptors became composite")
	}
}

func TestRiskRejectsMalformedInterface(t *testing.T) {
	_, err := AssessIdentityRisk(Identity{
		Interfaces: []string{"keyboard"},
	})

	if err == nil {
		t.Fatal("accepted malformed USB interface descriptor")
	}
}

func mustRisk(
	t *testing.T,
	identity Identity,
) []RiskFinding {
	t.Helper()

	findings, err := AssessIdentityRisk(identity)
	if err != nil {
		t.Fatal(err)
	}

	return findings
}

func requireRisk(
	t *testing.T,
	findings []RiskFinding,
	code string,
	severity RiskSeverity,
) {
	t.Helper()

	for _, finding := range findings {
		if finding.Code != code {
			continue
		}

		if finding.Severity != severity {
			t.Fatalf(
				"%s severity = %s, want %s",
				code,
				finding.Severity,
				severity,
			)
		}

		return
	}

	t.Fatalf("missing risk finding %q: %#v", code, findings)
}

func hasRisk(
	findings []RiskFinding,
	code string,
) bool {
	for _, finding := range findings {
		if finding.Code == code {
			return true
		}
	}

	return false
}
