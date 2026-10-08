package readmodel

import (
	"context"
	"errors"
	"testing"

	"github.com/JadeOpenServices/gjallarOS/internal/usbtrust"
)

type fakeResolved struct {
	value map[string]any
	err   error
	calls int
}

func (f *fakeResolved) Resolved(
	context.Context,
) (map[string]any, error) {
	f.calls++
	return f.value, f.err
}

type fakeObserved struct {
	value []usbtrust.ObservedDevice
	err   error
	calls int
}

func (f *fakeObserved) Observed(
	context.Context,
) ([]usbtrust.ObservedDevice, error) {
	f.calls++
	return f.value, f.err
}

type fakeTrust struct {
	value *usbtrust.Document
	err   error
	calls int
}

func (f *fakeTrust) Trusted(
	context.Context,
) (*usbtrust.Document, error) {
	f.calls++
	return f.value, f.err
}

func resolvedInternalUSB() map[string]any {
	return map[string]any{
		"hardware": map[string]any{
			"security": map[string]any{
				"fingerprint": map[string]any{
					"primary": map[string]any{
						"bus":        "usb",
						"attachment": "internal",
						"deviceId":   "1234:5678",
					},
				},
			},
		},
	}
}

func observedInternal() usbtrust.ObservedDevice {
	return usbtrust.ObservedDevice{
		RuntimeID: "7",
		Identity: usbtrust.Identity{
			VIDPID:      "1234:5678",
			Serial:      "synthetic-internal",
			Hash:        "synthetic-descriptor",
			ParentHash:  "synthetic-parent",
			Port:        "1-1",
			Interfaces:  []string{"ff:00:00"},
			ConnectType: "hardwired",
		},
	}
}

func trustedInternal() usbtrust.Device {
	observed := observedInternal()

	return usbtrust.Device{
		ID:             "internal:fingerprint",
		Role:           "hardware.security.fingerprint.primary",
		Class:          usbtrust.ClassInternal,
		Portable:       false,
		ExpectedByODDC: true,
		Strength:       usbtrust.StrengthSerialDescriptorTopology,
		Identity:       observed.Identity,
		FirstAccepted:  "2026-09-17T00:00:00Z",
		LastAccepted:   "2026-09-17T00:00:00Z",
	}
}

func trustedDocument() *usbtrust.Document {
	return &usbtrust.Document{
		Schema:    usbtrust.SchemaVersion,
		MachineID: "synthetic-machine",
		ODDCModel: "model/synthetic",
		Revision:  4,
		Devices: []usbtrust.Device{
			trustedInternal(),
		},
	}
}

func TestStatusComposesAllSources(t *testing.T) {
	resolved := &fakeResolved{
		value: resolvedInternalUSB(),
	}

	observed := &fakeObserved{
		value: []usbtrust.ObservedDevice{
			observedInternal(),
			{
				RuntimeID: "8",
				Identity: usbtrust.Identity{
					VIDPID:     "aaaa:0001",
					Hash:       "external-descriptor",
					Interfaces: []string{"08:06:50"},
				},
			},
		},
	}

	trust := &fakeTrust{
		value: trustedDocument(),
	}

	reader := Reader{
		Resolved:     resolved,
		Observations: observed,
		Trust:        trust,
	}

	status, err := reader.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if !status.StatePresent {
		t.Fatal("signed trust state reported absent")
	}

	if status.Revision != 4 {
		t.Fatalf(
			"revision = %d, want 4",
			status.Revision,
		)
	}

	if status.TrustedDevices != 1 {
		t.Fatalf(
			"trusted devices = %d, want 1",
			status.TrustedDevices,
		)
	}

	if status.ExpectedDevices != 1 {
		t.Fatalf(
			"expected devices = %d, want 1",
			status.ExpectedDevices,
		)
	}

	if status.ObservedDevices != 2 {
		t.Fatalf(
			"observed devices = %d, want 2",
			status.ObservedDevices,
		)
	}

	requireSingleCalls(t, resolved, observed, trust)
}

func TestAuditMatchesTrustedInternal(t *testing.T) {
	reader := Reader{
		Resolved: &fakeResolved{
			value: resolvedInternalUSB(),
		},
		Observations: &fakeObserved{
			value: []usbtrust.ObservedDevice{
				observedInternal(),
			},
		},
		Trust: &fakeTrust{
			value: trustedDocument(),
		},
	}

	result, revision, err := reader.Audit(
		context.Background(),
	)
	if err != nil {
		t.Fatal(err)
	}

	if revision != 4 {
		t.Fatalf(
			"revision = %d, want 4",
			revision,
		)
	}

	requireAuditFinding(
		t,
		result,
		usbtrust.CodeInternalMatch,
		usbtrust.AuditPass,
	)
}

func TestAuditWithoutStateRequiresEnrollmentReview(
	t *testing.T,
) {
	reader := Reader{
		Resolved: &fakeResolved{
			value: resolvedInternalUSB(),
		},
		Observations: &fakeObserved{
			value: []usbtrust.ObservedDevice{
				observedInternal(),
			},
		},
		Trust: &fakeTrust{},
	}

	result, revision, err := reader.Audit(
		context.Background(),
	)
	if err != nil {
		t.Fatal(err)
	}

	if revision != 0 {
		t.Fatalf(
			"unenrolled revision = %d, want 0",
			revision,
		)
	}

	requireAuditFinding(
		t,
		result,
		usbtrust.CodeInternalUnvalidated,
		usbtrust.AuditReview,
	)
}

func TestStatusWithoutTrustStateIsValid(t *testing.T) {
	reader := Reader{
		Resolved: &fakeResolved{
			value: resolvedInternalUSB(),
		},
		Observations: &fakeObserved{
			value: []usbtrust.ObservedDevice{
				observedInternal(),
			},
		},
		Trust: &fakeTrust{},
	}

	status, err := reader.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if status.StatePresent {
		t.Fatal("unenrolled machine reported trust state")
	}

	if status.Revision != 0 {
		t.Fatalf(
			"revision = %d, want 0",
			status.Revision,
		)
	}
}

func TestReaderFailsClosedOnObservationFailure(
	t *testing.T,
) {
	reader := Reader{
		Resolved: &fakeResolved{
			value: resolvedInternalUSB(),
		},
		Observations: &fakeObserved{
			err: errors.New("synthetic observation failure"),
		},
		Trust: &fakeTrust{
			value: trustedDocument(),
		},
	}

	if _, err := reader.Status(
		context.Background(),
	); err == nil {
		t.Fatal("observation failure became status success")
	}
}

func TestReaderRejectsInvalidTrustedState(t *testing.T) {
	invalid := trustedDocument()
	invalid.Revision = 0

	reader := Reader{
		Resolved: &fakeResolved{
			value: resolvedInternalUSB(),
		},
		Observations: &fakeObserved{
			value: []usbtrust.ObservedDevice{
				observedInternal(),
			},
		},
		Trust: &fakeTrust{
			value: invalid,
		},
	}

	if _, err := reader.Status(
		context.Background(),
	); err == nil {
		t.Fatal("invalid authoritative state became status success")
	}
}

func requireSingleCalls(
	t *testing.T,
	resolved *fakeResolved,
	observed *fakeObserved,
	trust *fakeTrust,
) {
	t.Helper()

	if resolved.calls != 1 {
		t.Fatalf(
			"resolved calls = %d, want 1",
			resolved.calls,
		)
	}

	if observed.calls != 1 {
		t.Fatalf(
			"observation calls = %d, want 1",
			observed.calls,
		)
	}

	if trust.calls != 1 {
		t.Fatalf(
			"trust calls = %d, want 1",
			trust.calls,
		)
	}
}

func requireAuditFinding(
	t *testing.T,
	result usbtrust.AuditResult,
	code usbtrust.AuditCode,
	state usbtrust.AuditState,
) {
	t.Helper()

	for _, finding := range result.Findings {
		if finding.Code != code {
			continue
		}

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

	t.Fatalf(
		"missing %s finding in %#v",
		code,
		result.Findings,
	)
}
