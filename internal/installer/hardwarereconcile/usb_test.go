package hardwarereconcile

import (
	"testing"

	"github.com/JadeOpenServices/gjallarOS/internal/hardware/deviceprobe"
	"github.com/JadeOpenServices/gjallarOS/internal/usbtrust"
)

func internalUSB(deviceID string, required bool) map[string]any {
	return map[string]any{
		"hardware": map[string]any{
			"security": map[string]any{
				"fingerprint": map[string]any{
					"primary": map[string]any{
						"bus":        "usb",
						"attachment": "internal",
						"deviceId":   deviceID,
						"required":   required,
					},
				},
			},
		},
	}
}

func snapshotUSB(devices ...deviceprobe.USBDevice) deviceprobe.Snapshot {
	return deviceprobe.Snapshot{
		Schema:     1,
		USBDevices: devices,
	}
}

func TestInternalUSBUniqueCandidateNeedsReview(t *testing.T) {
	got, err := InternalUSB(
		internalUSB("27c6:609c", true),
		snapshotUSB(deviceprobe.USBDevice{
			Path:    "1-4",
			Vendor:  "27c6",
			Product: "609c",
		}),
	)
	if err != nil {
		t.Fatal(err)
	}

	requireFinding(
		t,
		got,
		usbtrust.CodeInternalUnvalidated,
		usbtrust.AuditReview,
	)
}

func TestInternalUSBMissingRequiredDeviceWarns(t *testing.T) {
	got, err := InternalUSB(
		internalUSB("27c6:609c", true),
		snapshotUSB(),
	)
	if err != nil {
		t.Fatal(err)
	}

	requireFinding(
		t,
		got,
		usbtrust.CodeInternalMissing,
		usbtrust.AuditWarn,
	)
}

func TestInternalUSBAmbiguousCandidatesNeedReview(t *testing.T) {
	got, err := InternalUSB(
		internalUSB("27c6:609c", true),
		snapshotUSB(
			deviceprobe.USBDevice{
				Path:    "1-4",
				Vendor:  "27c6",
				Product: "609c",
			},
			deviceprobe.USBDevice{
				Path:    "3-2",
				Vendor:  "27c6",
				Product: "609c",
			},
		),
	)
	if err != nil {
		t.Fatal(err)
	}

	count := 0

	for _, finding := range got {
		if finding.Code == usbtrust.CodeInternalAmbiguous &&
			finding.State == usbtrust.AuditReview {
			count++
		}
	}

	if count != 2 {
		t.Fatalf(
			"ambiguous findings = %d, want 2: %#v",
			count,
			got,
		)
	}
}

func TestInternalUSBIgnoresUnrelatedObservedUSB(t *testing.T) {
	got, err := InternalUSB(
		internalUSB("27c6:609c", true),
		snapshotUSB(
			deviceprobe.USBDevice{
				Path:    "1-4",
				Vendor:  "27c6",
				Product: "609c",
			},
			deviceprobe.USBDevice{
				Path:    "2-1",
				Vendor:  "abcd",
				Product: "1234",
			},
		),
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 1 {
		t.Fatalf(
			"ODDC internal findings = %d, want 1: %#v",
			len(got),
			got,
		)
	}

	if got[0].Code != usbtrust.CodeInternalUnvalidated {
		t.Fatalf("unexpected finding: %+v", got[0])
	}
}

func TestInternalUSBOptionalMissingDeviceIsNotAProblem(t *testing.T) {
	got, err := InternalUSB(
		internalUSB("1234:5678", false),
		snapshotUSB(),
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 0 {
		t.Fatalf("optional missing device produced findings: %#v", got)
	}
}

func requireFinding(
	t *testing.T,
	findings []usbtrust.Finding,
	code usbtrust.AuditCode,
	state usbtrust.AuditState,
) {
	t.Helper()

	for _, finding := range findings {
		if finding.Code == code && finding.State == state {
			return
		}
	}

	t.Fatalf(
		"missing %s/%s finding in %#v",
		code,
		state,
		findings,
	)
}
