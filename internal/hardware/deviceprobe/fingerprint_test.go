package deviceprobe

import (
	"reflect"
	"testing"
)

func TestParseFingerprintDevicePaths(t *testing.T) {
	got, err := parseObjectPaths(
		`ao 2 "/net/reactivated/Fprint/Device/0" "/net/reactivated/Fprint/Device/1"`,
	)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"/net/reactivated/Fprint/Device/0",
		"/net/reactivated/Fprint/Device/1",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestParseNoFingerprintDevices(t *testing.T) {
	got, err := parseObjectPaths("ao 0")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("expected non-nil empty device list, got %#v", got)
	}
}

func TestParseFingerprintName(t *testing.T) {
	got := parseBusctlString(`s "Goodix Fingerprint Device"`)
	if got != "Goodix Fingerprint Device" {
		t.Fatalf("unexpected device name %q", got)
	}
}
