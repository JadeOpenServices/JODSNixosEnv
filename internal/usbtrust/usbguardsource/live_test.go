package usbguardsource

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/JadeOpenServices/gjallarOS/internal/usbtrust"
)

type fakeRunner struct {
	output []byte
	err    error
	name   string
	args   []string
	calls  int
}

func (f *fakeRunner) Output(
	_ context.Context,
	name string,
	args ...string,
) ([]byte, error) {
	f.calls++
	f.name = name
	f.args = append([]string(nil), args...)

	return f.output, f.err
}

func TestLiveSourceReadsRuntimeDevicesAndFiltersControllers(
	t *testing.T,
) {
	runner := &fakeRunner{
		output: []byte(
			`1: allow id aaaa:0001 serial "root" name "Root Hub" hash "root-hash" parent-hash "root-parent" via-port "usb1" with-interface 09:00:00 with-connect-type ""
2: allow id bbbb:0001 serial "internal" name "Internal Device" hash "internal-hash" parent-hash "root-hash" via-port "1-4" with-interface ff:00:00 with-connect-type "hardwired"
3: block id cccc:0001 serial "external" name "External HID" hash "external-hash" parent-hash "dock-hash" via-port "7-1.1" with-interface 03:01:01 with-connect-type "unknown"
`,
		),
	}

	source := LiveSource{
		Runner: runner,
		Binary: "/synthetic/usbguard",
	}

	got, err := source.Observed(
		context.Background(),
	)
	if err != nil {
		t.Fatal(err)
	}

	if runner.calls != 1 {
		t.Fatalf(
			"runner calls = %d, want 1",
			runner.calls,
		)
	}

	if runner.name != "/synthetic/usbguard" {
		t.Fatalf(
			"binary = %q",
			runner.name,
		)
	}

	if !reflect.DeepEqual(
		runner.args,
		[]string{"list-devices"},
	) {
		t.Fatalf(
			"args = %#v",
			runner.args,
		)
	}

	if len(got) != 2 {
		t.Fatalf(
			"observed devices = %d, want 2: %#v",
			len(got),
			got,
		)
	}

	for _, device := range got {
		if device.RuntimeID == "1" {
			t.Fatal(
				"host-controller root hub leaked into observations",
			)
		}
	}
}

func TestHotplugHubIsNotFilteredAsController(
	t *testing.T,
) {
	runner := &fakeRunner{
		output: []byte(
			`7: block id aaaa:0002 serial "" name "Dock Hub" hash "dock-hub" parent-hash "parent" via-port "7-1" with-interface 09:00:00 with-connect-type "hotplug"`,
		),
	}

	got, err := (LiveSource{
		Runner: runner,
	}).Observed(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 1 {
		t.Fatalf(
			"hotplug hub observations = %d, want 1",
			len(got),
		)
	}
}

func TestNestedUnknownHubIsNotFilteredAsController(
	t *testing.T,
) {
	runner := &fakeRunner{
		output: []byte(
			`8: block id aaaa:0003 serial "" name "Nested Hub" hash "nested-hub" parent-hash "parent" via-port "7-1.1" with-interface 09:00:00 with-connect-type "unknown"`,
		),
	}

	got, err := (LiveSource{
		Runner: runner,
	}).Observed(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 1 {
		t.Fatalf(
			"nested hub observations = %d, want 1",
			len(got),
		)
	}
}

func TestLiveSourceRejectsNonRuntimePolicyOutput(
	t *testing.T,
) {
	runner := &fakeRunner{
		output: []byte(
			`allow id aaaa:0001 serial "x" name "Device" hash "hash" parent-hash "parent" via-port "1-1" with-interface ff:00:00 with-connect-type "hardwired"`,
		),
	}

	_, err := (LiveSource{
		Runner: runner,
	}).Observed(context.Background())

	if err == nil {
		t.Fatal(
			"live source accepted observation without runtime id",
		)
	}
}

func TestLiveSourcePropagatesRunnerFailure(
	t *testing.T,
) {
	runner := &fakeRunner{
		err: errors.New("synthetic daemon failure"),
	}

	_, err := (LiveSource{
		Runner: runner,
	}).Observed(context.Background())

	if err == nil {
		t.Fatal("USBGuard runner failure became success")
	}

	if !strings.Contains(
		err.Error(),
		"synthetic daemon failure",
	) {
		t.Fatalf(
			"unexpected error %q",
			err,
		)
	}
}

func TestControllerFilterDoesNotUseDeviceIdentity(
	t *testing.T,
) {
	tests := []struct {
		name     string
		port     string
		connect  string
		ifaces   []string
		filtered bool
	}{
		{
			name:     "root hub shape",
			port:     "usb8",
			ifaces:   []string{"09:00:00"},
			filtered: true,
		},
		{
			name:     "root composite hub shape",
			port:     "usb8",
			ifaces:   []string{"09:00:01", "09:00:02"},
			filtered: true,
		},
		{
			name:     "external port",
			port:     "8-1",
			ifaces:   []string{"09:00:00"},
			filtered: false,
		},
		{
			name:     "hotplug root-shaped device",
			port:     "usb8",
			connect:  "hotplug",
			ifaces:   []string{"09:00:00"},
			filtered: false,
		},
		{
			name:     "non-hub root port",
			port:     "usb8",
			ifaces:   []string{"03:01:01"},
			filtered: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := controllerInfrastructure(
				identityForFilter(
					test.port,
					test.connect,
					test.ifaces,
				),
			)

			if got != test.filtered {
				t.Fatalf(
					"filtered = %t, want %t",
					got,
					test.filtered,
				)
			}
		})
	}
}

func identityForFilter(
	port string,
	connect string,
	ifaces []string,
) usbtrust.Identity {
	return usbtrust.Identity{
		VIDPID:      "ffff:ffff",
		Hash:        "synthetic",
		Port:        port,
		ConnectType: connect,
		Interfaces:  ifaces,
	}
}
