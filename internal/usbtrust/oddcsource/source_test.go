package oddcsource

import (
	"reflect"
	"testing"

	"github.com/bakanura/gjallarOS/pkg/oddc"
)

func TestExpectedFiltersTransportAndAttachment(t *testing.T) {
	resolved := map[string]any{
		"hardware": map[string]any{
			"security": map[string]any{
				"fingerprint": map[string]any{
					"primary": map[string]any{
						"bus":        "usb",
						"attachment": "internal",
						"deviceId":   "27C6:609C",
					},
				},
			},
			"input": map[string]any{
				"touchpad": map[string]any{
					"primary": map[string]any{
						"bus":        "i2c",
						"attachment": "internal",
						"deviceId":   "093a:0274",
					},
				},
			},
			"expansion": map[string]any{
				"ethernet": map[string]any{
					"primary": map[string]any{
						"bus":        "usb",
						"attachment": "modular",
						"deviceId":   "0bda:8156",
					},
				},
			},
		},
	}

	got, err := Expected(resolved)
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 1 {
		t.Fatalf(
			"got %d expectations, want 1: %#v",
			len(got),
			got,
		)
	}

	if got[0].Role != "hardware.security.fingerprint.primary" {
		t.Fatalf("unexpected role %q", got[0].Role)
	}

	if got[0].VIDPID != "27c6:609c" {
		t.Fatalf("unexpected VID:PID %q", got[0].VIDPID)
	}

	if !got[0].Required {
		t.Fatal("internal component should default to required")
	}
}

func TestExpectedSupportsOptionalInternalDevice(t *testing.T) {
	resolved := map[string]any{
		"hardware": map[string]any{
			"camera": map[string]any{
				"primary": map[string]any{
					"bus":        "usb",
					"attachment": "internal",
					"deviceId":   "1234:5678",
					"required":   false,
				},
			},
		},
	}

	got, err := Expected(resolved)
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 1 {
		t.Fatalf("got %d expectations, want 1", len(got))
	}

	if got[0].Required {
		t.Fatal("optional internal device became required")
	}
}

func TestExpectedFailsClosedWithoutDeviceID(t *testing.T) {
	resolved := map[string]any{
		"hardware": map[string]any{
			"security": map[string]any{
				"primary": map[string]any{
					"bus":        "usb",
					"attachment": "internal",
				},
			},
		},
	}

	if _, err := Expected(resolved); err == nil {
		t.Fatal("accepted internal USB expectation without deviceId")
	}
}

func TestFrameworkResolvedUSBExpectations(t *testing.T) {
	registry, err := oddc.LoadRegistry("../../../oddc")
	if err != nil {
		t.Fatal(err)
	}

	resolved, err := registry.ResolveModel(
		"model/framework/laptop-13-amd-ryzen-7040",
		nil,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	got, err := Expected(resolved.Resolved)
	if err != nil {
		t.Fatal(err)
	}

	roles := make([]string, 0, len(got))

	for _, expected := range got {
		roles = append(roles, expected.Role)
	}

	want := []string{
		"hardware.network.bluetooth.primary",
		"hardware.security.fingerprint.primary",
	}

	if !reflect.DeepEqual(roles, want) {
		t.Fatalf(
			"Framework internal USB roles = %#v, want %#v",
			roles,
			want,
		)
	}
}
