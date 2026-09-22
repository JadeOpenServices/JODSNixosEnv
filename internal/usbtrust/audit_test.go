package usbtrust

import "testing"

func internalExpectation() ExpectedDevice {
	return ExpectedDevice{
		Role:        "fingerprint",
		VIDPID:      "27c6:609c",
		ConnectType: "hardwired",
		Interfaces:  []string{"ff:00:00"},
		Required:    true,
	}
}

func internalTrusted() Device {
	return Device{
		ID:             "internal:fingerprint",
		Role:           "fingerprint",
		Class:          ClassInternal,
		ExpectedByODDC: true,
		Portable:       false,
		Strength:       StrengthSerialDescriptorTopology,
		Identity: Identity{
			VIDPID:      "27c6:609c",
			Serial:      "GOODIX-A",
			Hash:        "descriptor-a",
			ParentHash:  "parent-a",
			Interfaces:  []string{"ff:00:00"},
			ConnectType: "hardwired",
		},
		FirstAccepted: "2026-09-17T00:00:00Z",
		LastAccepted:  "2026-09-17T00:00:00Z",
	}
}

func trustedDocument(devices ...Device) *Document {
	return &Document{
		Schema:    SchemaVersion,
		MachineID: "machine-test",
		ODDCModel: "model/test",
		Revision:  1,
		Devices:   devices,
	}
}

func TestAuditInternalMatch(t *testing.T) {
	trusted := internalTrusted()

	result, err := Audit(AuditInput{
		Expected: []ExpectedDevice{
			internalExpectation(),
		},
		Trusted: trustedDocument(trusted),
		Observed: []ObservedDevice{
			{
				RuntimeID: "7",
				Identity:  trusted.Identity,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	requireFinding(t, result, CodeInternalMatch, AuditPass)
}

func TestAuditDetectsInternalReplacement(t *testing.T) {
	trusted := internalTrusted()

	replacement := trusted.Identity
	replacement.Hash = "different-descriptor"
	replacement.Serial = "GOODIX-B"

	result, err := Audit(AuditInput{
		Expected: []ExpectedDevice{
			internalExpectation(),
		},
		Trusted: trustedDocument(trusted),
		Observed: []ObservedDevice{
			{
				RuntimeID: "8",
				Identity:  replacement,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	requireFinding(t, result, CodeInternalChanged, AuditBlock)
}

func TestAuditDetectsTopologyChange(t *testing.T) {
	trusted := internalTrusted()

	moved := trusted.Identity
	moved.ParentHash = "different-parent"

	result, err := Audit(AuditInput{
		Expected: []ExpectedDevice{
			internalExpectation(),
		},
		Trusted: trustedDocument(trusted),
		Observed: []ObservedDevice{
			{
				RuntimeID: "9",
				Identity:  moved,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	requireFinding(t, result, CodeInternalChanged, AuditBlock)
}

func TestAuditInitialInternalNeedsReview(t *testing.T) {
	observed := internalTrusted().Identity

	result, err := Audit(AuditInput{
		Expected: []ExpectedDevice{
			internalExpectation(),
		},
		Observed: []ObservedDevice{
			{
				RuntimeID: "10",
				Identity:  observed,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	requireFinding(
		t,
		result,
		CodeInternalUnvalidated,
		AuditReview,
	)
}

func TestAuditMissingInternalWarns(t *testing.T) {
	result, err := Audit(AuditInput{
		Expected: []ExpectedDevice{
			internalExpectation(),
		},
		Trusted: trustedDocument(internalTrusted()),
	})
	if err != nil {
		t.Fatal(err)
	}

	requireFinding(t, result, CodeInternalMissing, AuditWarn)
}

func TestAuditUnexpectedHardwiredBlocks(t *testing.T) {
	result, err := Audit(AuditInput{
		Observed: []ObservedDevice{
			{
				RuntimeID: "11",
				Identity: Identity{
					VIDPID:      "1234:5678",
					Hash:        "unknown",
					ConnectType: "hardwired",
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	requireFinding(
		t,
		result,
		CodeUnexpectedInternal,
		AuditBlock,
	)
}

func TestAuditUnknownExternalBlocks(t *testing.T) {
	result, err := Audit(AuditInput{
		Observed: []ObservedDevice{
			{
				RuntimeID: "12",
				Identity: Identity{
					VIDPID: "abcd:1234",
					Hash:   "external",
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	requireFinding(
		t,
		result,
		CodeUnknownExternal,
		AuditBlock,
	)
}

func TestPortableExternalMayMoveTopology(t *testing.T) {
	external := Device{
		ID:       "external:mouse",
		Class:    ClassExternal,
		Portable: true,
		Strength: StrengthSerialDescriptor,
		Identity: Identity{
			VIDPID:      "046d:c548",
			Serial:      "mouse-1",
			Hash:        "mouse-descriptor",
			ParentHash:  "old-parent",
			Port:        "7-1.2",
			Interfaces:  []string{"03:01:02"},
			ConnectType: "hotplug",
		},
		FirstAccepted: "2026-09-17T00:00:00Z",
		LastAccepted:  "2026-09-17T00:00:00Z",
	}

	observed := external.Identity
	observed.ParentHash = "new-parent"
	observed.Port = "8-1.4"

	result, err := Audit(AuditInput{
		Trusted: trustedDocument(external),
		Observed: []ObservedDevice{
			{
				RuntimeID: "13",
				Identity:  observed,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	requireFinding(
		t,
		result,
		CodeTrustedExternal,
		AuditPass,
	)
}

func requireFinding(
	t *testing.T,
	result AuditResult,
	code AuditCode,
	state AuditState,
) {
	t.Helper()

	for _, finding := range result.Findings {
		if finding.Code == code {
			if finding.State != state {
				t.Fatalf(
					"%s state = %s, want %s",
					code,
					finding.State,
					state,
				)
			}

			return
		}
	}

	t.Fatalf(
		"missing finding %s in %#v",
		code,
		result.Findings,
	)
}
