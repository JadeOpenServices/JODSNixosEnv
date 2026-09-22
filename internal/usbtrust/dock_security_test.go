package usbtrust

import "testing"

func syntheticDock() Device {
	return Device{
		ID:       "external:dock",
		Class:    ClassExternal,
		Portable: false,
		Strength: StrengthSerialDescriptorTopology,
		Identity: Identity{
			VIDPID:      "1111:0001",
			Serial:      "synthetic-dock",
			Hash:        "synthetic-dock-hash",
			ParentHash:  "synthetic-root",
			Port:        "7-1",
			Interfaces:  []string{"09:00:00"},
			ConnectType: "hotplug",
		},
		FirstAccepted: "2026-09-17T00:00:00Z",
		LastAccepted:  "2026-09-17T00:00:00Z",
	}
}

func TestTrustedDockDoesNotTrustChildren(t *testing.T) {
	dock := syntheticDock()

	storage := ObservedDevice{
		RuntimeID: "21",
		Identity: Identity{
			VIDPID:      "2222:0001",
			Serial:      "synthetic-storage",
			Hash:        "synthetic-storage-hash",
			ParentHash:  dock.Identity.Hash,
			Port:        "7-1.1",
			Interfaces:  []string{"08:06:50"},
			ConnectType: "unknown",
		},
	}

	hid := ObservedDevice{
		RuntimeID: "22",
		Identity: Identity{
			VIDPID:      "3333:0001",
			Serial:      "synthetic-hid",
			Hash:        "synthetic-hid-hash",
			ParentHash:  dock.Identity.Hash,
			Port:        "7-1.2",
			Interfaces:  []string{"03:01:01"},
			ConnectType: "unknown",
		},
	}

	network := ObservedDevice{
		RuntimeID: "23",
		Identity: Identity{
			VIDPID:     "4444:0001",
			Serial:     "synthetic-network",
			Hash:       "synthetic-network-hash",
			ParentHash: dock.Identity.Hash,
			Port:       "7-1.3",
			Interfaces: []string{
				"02:06:00",
				"0a:00:00",
			},
			ConnectType: "unknown",
		},
	}

	result, err := Audit(AuditInput{
		Trusted: trustedDocument(dock),
		Observed: []ObservedDevice{
			{
				RuntimeID: "20",
				Identity:  dock.Identity,
			},
			storage,
			hid,
			network,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if got := countFinding(
		result,
		CodeTrustedExternal,
		AuditPass,
	); got != 1 {
		t.Fatalf(
			"trusted dock PASS findings = %d, want 1",
			got,
		)
	}

	if got := countFinding(
		result,
		CodeUnknownExternal,
		AuditBlock,
	); got != 3 {
		t.Fatalf(
			"unknown dock-child BLOCK findings = %d, want 3",
			got,
		)
	}

	storageRisk, err := AssessIdentityRisk(storage.Identity)
	if err != nil {
		t.Fatal(err)
	}

	requireDockRisk(
		t,
		storageRisk,
		RiskMassStorage,
		RiskHigh,
	)

	hidRisk, err := AssessIdentityRisk(hid.Identity)
	if err != nil {
		t.Fatal(err)
	}

	requireDockRisk(
		t,
		hidRisk,
		RiskHIDInput,
		RiskHigh,
	)

	networkRisk, err := AssessIdentityRisk(network.Identity)
	if err != nil {
		t.Fatal(err)
	}

	requireDockRisk(
		t,
		networkRisk,
		RiskNetwork,
		RiskHigh,
	)
}

func TestTrustedDockChildInterfaceMutationBlocks(t *testing.T) {
	child := Device{
		ID:       "external:dock-network",
		Class:    ClassExternal,
		Portable: false,
		Strength: StrengthSerialDescriptorTopology,
		Identity: Identity{
			VIDPID:     "5555:0001",
			Serial:     "synthetic-network-child",
			Hash:       "synthetic-network-child-hash",
			ParentHash: "synthetic-dock-hash",
			Port:       "7-1.1",
			Interfaces: []string{
				"02:06:00",
				"0a:00:00",
			},
			ConnectType: "unknown",
		},
		FirstAccepted: "2026-09-17T00:00:00Z",
		LastAccepted:  "2026-09-17T00:00:00Z",
	}

	changed := child.Identity

	changed.Interfaces = []string{
		"02:06:00",
		"0a:00:00",
		"03:01:01",
	}

	result, err := Audit(AuditInput{
		Trusted: trustedDocument(child),
		Observed: []ObservedDevice{
			{
				RuntimeID: "31",
				Identity:  changed,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if got := countFinding(
		result,
		CodeTrustedExternal,
		AuditPass,
	); got != 0 {
		t.Fatalf(
			"mutated child retained trust: %d PASS findings",
			got,
		)
	}

	if got := countFinding(
		result,
		CodeTrustedExternalChanged,
		AuditBlock,
	); got != 1 {
		t.Fatalf(
			"mutated trusted-child BLOCK findings = %d, want 1",
			got,
		)
	}

	findings, err := AssessIdentityRisk(changed)
	if err != nil {
		t.Fatal(err)
	}

	requireDockRisk(
		t,
		findings,
		RiskHIDInput,
		RiskHigh,
	)

	requireDockRisk(
		t,
		findings,
		RiskNetwork,
		RiskHigh,
	)

	requireDockRisk(
		t,
		findings,
		RiskHIDNetworkComposite,
		RiskHigh,
	)
}

func countFinding(
	result AuditResult,
	code AuditCode,
	state AuditState,
) int {
	count := 0

	for _, finding := range result.Findings {
		if finding.Code == code &&
			finding.State == state {
			count++
		}
	}

	return count
}

func requireDockRisk(
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

	t.Fatalf(
		"missing risk %q in %#v",
		code,
		findings,
	)
}

func TestSameModelDifferentSerialRemainsUnknown(t *testing.T) {
	trusted := Device{
		ID:       "external:known-device",
		Class:    ClassExternal,
		Portable: true,
		Strength: StrengthSerialDescriptor,
		Identity: Identity{
			VIDPID:     "6666:0001",
			Serial:     "accepted-serial",
			Hash:       "accepted-hash",
			Interfaces: []string{"03:01:02"},
		},
		FirstAccepted: "2026-09-17T00:00:00Z",
		LastAccepted:  "2026-09-17T00:00:00Z",
	}

	result, err := Audit(AuditInput{
		Trusted: trustedDocument(trusted),
		Observed: []ObservedDevice{
			{
				RuntimeID: "41",
				Identity: Identity{
					VIDPID:     "6666:0001",
					Serial:     "different-serial",
					Hash:       "different-hash",
					Interfaces: []string{"03:01:02"},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if got := countFinding(
		result,
		CodeUnknownExternal,
		AuditBlock,
	); got != 1 {
		t.Fatalf(
			"same-model different device findings = %d, want unknown BLOCK",
			got,
		)
	}

	if got := countFinding(
		result,
		CodeTrustedExternalChanged,
		AuditBlock,
	); got != 0 {
		t.Fatalf(
			"different serial falsely correlated with trusted device",
		)
	}
}
