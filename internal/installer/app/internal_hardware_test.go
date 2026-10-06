package app

import (
	"context"
	"strings"
	"testing"

	portable "github.com/JadeOpenServices/oddc"
	"github.com/bakanura/gjallarOS/internal/hardware/deviceprobe"
	"github.com/bakanura/gjallarOS/internal/installer/discovery"
	"github.com/bakanura/gjallarOS/internal/installer/oddc"
	"github.com/bakanura/gjallarOS/internal/installer/prompt"
)

const internalUSBTestModel = "model/test/laptop"

func resolvedInternalUSBForTest() oddc.Resolved {
	return oddc.Resolved{
		ModelID: internalUSBTestModel,
		Canonical: portable.Resolved{
			ModelID: internalUSBTestModel,
			Resolved: map[string]any{
				"hardware": map[string]any{
					"security": map[string]any{
						"fingerprint": map[string]any{
							"primary": map[string]any{
								"bus":        "usb",
								"attachment": "internal",
								"deviceId":   "27c6:609c",
							},
						},
					},
				},
			},
		},
	}
}

func installerHardwareWithUSB(paths ...string) discovery.Hardware {
	snapshot := deviceprobe.Snapshot{
		USBDevices: []deviceprobe.USBDevice{},
	}

	for _, path := range paths {
		snapshot.USBDevices = append(
			snapshot.USBDevices,
			deviceprobe.USBDevice{
				Path:    path,
				Vendor:  "27c6",
				Product: "609c",
			},
		)
	}

	return discovery.Hardware{
		DeviceInventory: snapshot,
	}
}

func TestReconcileInternalHardwareRescansMissingDevice(t *testing.T) {
	previous := detectInstallerHardware
	defer func() {
		detectInstallerHardware = previous
	}()

	detectInstallerHardware = func(string) discovery.Hardware {
		return installerHardwareWithUSB("1-1")
	}

	var output strings.Builder
	ui := prompt.New(
		strings.NewReader("y\n"),
		&output,
	)

	hardware, host, changed, err := reconcileInternalHardware(
		context.Background(),
		ui,
		resolvedInternalUSBForTest(),
		installerHardwareWithUSB(),
		nil,
		false,
		&output,
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(hardware.DeviceInventory.USBDevices) != 1 {
		t.Fatalf(
			"rescanned USB devices = %d, want 1",
			len(hardware.DeviceInventory.USBDevices),
		)
	}
	if host != nil || changed {
		t.Fatal("successful rescan unexpectedly staged host state")
	}
}

func TestReconcileInternalHardwareTemporaryAbsenceKeepsCanonicalReference(
	t *testing.T,
) {
	var output strings.Builder
	ui := prompt.New(
		strings.NewReader("n\nn\n"),
		&output,
	)

	resolved := resolvedInternalUSBForTest()

	_, host, changed, err := reconcileInternalHardware(
		context.Background(),
		ui,
		resolved,
		installerHardwareWithUSB(),
		nil,
		false,
		&output,
	)
	if err != nil {
		t.Fatal(err)
	}

	if host != nil || changed {
		t.Fatal("temporary absence unexpectedly changed host ODDC state")
	}

	if _, exists := portable.Lookup(
		resolved.Canonical.Resolved,
		"hardware.security.fingerprint.primary",
	); !exists {
		t.Fatal("temporary absence removed canonical ODDC reference")
	}

	if !strings.Contains(
		output.String(),
		"temporarily absent",
	) {
		t.Fatalf(
			"temporary-absence warning missing:\n%s",
			output.String(),
		)
	}
}

func TestReconcileInternalHardwareStagesPermanentRemoval(t *testing.T) {
	var output strings.Builder
	ui := prompt.New(
		strings.NewReader("n\ny\n"),
		&output,
	)

	_, host, changed, err := reconcileInternalHardware(
		context.Background(),
		ui,
		resolvedInternalUSBForTest(),
		installerHardwareWithUSB(),
		nil,
		false,
		&output,
	)
	if err != nil {
		t.Fatal(err)
	}

	if host == nil {
		t.Fatal("permanent removal did not create staged host overlay")
	}
	if !changed {
		t.Fatal("permanent removal did not mark host state changed")
	}

	value, exists := portable.Lookup(
		host.Overrides,
		"hardware.security.fingerprint.primary.$delete",
	)
	if !exists || value != true {
		t.Fatalf(
			"staged deletion missing: %#v",
			host.Overrides,
		)
	}
}

func TestReconcileInternalHardwareAddsToExistingHostOverlay(t *testing.T) {
	host := oddc.HostOverlay{
		APIVersion:  portable.EntityAPIVersion,
		ID:          "host/machine-local",
		Kind:        "host",
		TargetModel: internalUSBTestModel,
		Overrides: map[string]any{
			"policy": map[string]any{
				"thermal": map[string]any{
					"profile": "quiet",
				},
			},
		},
	}

	var output strings.Builder
	ui := prompt.New(
		strings.NewReader("n\ny\n"),
		&output,
	)

	_, got, changed, err := reconcileInternalHardware(
		context.Background(),
		ui,
		resolvedInternalUSBForTest(),
		installerHardwareWithUSB(),
		&host,
		false,
		&output,
	)
	if err != nil {
		t.Fatal(err)
	}

	if got == nil || !changed {
		t.Fatal("existing host overlay was not updated")
	}

	value, exists := portable.Lookup(
		got.Overrides,
		"policy.thermal.profile",
	)
	if !exists || value != "quiet" {
		t.Fatalf(
			"unrelated existing host override lost: %#v",
			got.Overrides,
		)
	}
}

func TestReconcileInternalHardwareAmbiguityCannotBecomeRemoval(
	t *testing.T,
) {
	var output strings.Builder
	ui := prompt.New(
		strings.NewReader("n\n"),
		&output,
	)

	_, host, changed, err := reconcileInternalHardware(
		context.Background(),
		ui,
		resolvedInternalUSBForTest(),
		installerHardwareWithUSB("1-1", "2-1"),
		nil,
		false,
		&output,
	)
	if err == nil {
		t.Fatal("ambiguous internal hardware was allowed to continue")
	}
	if host != nil || changed {
		t.Fatal("ambiguous hardware incorrectly staged a removal")
	}
}

func TestReconcileInternalHardwareUnattendedFailsWithoutPrompt(
	t *testing.T,
) {
	var output strings.Builder
	ui := prompt.New(
		strings.NewReader(""),
		&output,
	)

	_, host, changed, err := reconcileInternalHardware(
		context.Background(),
		ui,
		resolvedInternalUSBForTest(),
		installerHardwareWithUSB(),
		nil,
		true,
		&output,
	)
	if err == nil {
		t.Fatal("unattended missing internal hardware was accepted")
	}
	if host != nil || changed {
		t.Fatal("unattended failure staged host state")
	}
}

func TestReconcileInternalHardwareUniqueCandidateNeedsNoPrompt(
	t *testing.T,
) {
	var output strings.Builder
	ui := prompt.New(
		strings.NewReader(""),
		&output,
	)

	_, host, changed, err := reconcileInternalHardware(
		context.Background(),
		ui,
		resolvedInternalUSBForTest(),
		installerHardwareWithUSB("1-1"),
		nil,
		false,
		&output,
	)
	if err != nil {
		t.Fatal(err)
	}
	if host != nil || changed {
		t.Fatal("identified internal hardware unexpectedly changed host state")
	}
}
