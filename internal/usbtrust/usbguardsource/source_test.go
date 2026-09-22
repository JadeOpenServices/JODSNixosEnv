package usbguardsource

import (
	"reflect"
	"testing"
)

func TestParseGeneratedInternalDevice(t *testing.T) {
	line := `allow id 27c6:609c serial "UID493B7C27_XXXX_MOC_B0" name "Goodix Fingerprint USB Device" hash "V0zPPWsGm5jHRAjECGaEYkb8nK0ohMduSEVELsLlZ0A=" parent-hash "p9XdYtshh7dWZvylwnT2mODxf1un62IZMB3s1EZjqbs=" via-port "1-4" with-interface ff:00:00 with-connect-type "hardwired"`

	got, err := ParseLine(line)
	if err != nil {
		t.Fatal(err)
	}

	if got.Target != "allow" {
		t.Fatalf("target = %q, want allow", got.Target)
	}

	if got.Device.RuntimeID != "" {
		t.Fatalf(
			"generated rule has runtime id %q",
			got.Device.RuntimeID,
		)
	}

	identity := got.Device.Identity

	if identity.VIDPID != "27c6:609c" {
		t.Fatalf("VID:PID = %q", identity.VIDPID)
	}

	if identity.Port != "1-4" {
		t.Fatalf("port = %q", identity.Port)
	}

	if identity.ConnectType != "hardwired" {
		t.Fatalf(
			"connect type = %q",
			identity.ConnectType,
		)
	}

	if !reflect.DeepEqual(
		identity.Interfaces,
		[]string{"ff:00:00"},
	) {
		t.Fatalf(
			"interfaces = %#v",
			identity.Interfaces,
		)
	}
}

func TestParseRuntimeBlockedDevice(t *testing.T) {
	line := `31: block id abcd:1234 serial "\xd0\x89" name "UDisk" hash "WwLIxRVjks6yVS9HHhP1Vn3rryx6WklV8RXeIlNoXag=" parent-hash "kv3v2+rnq9QvYI3/HbJ1EV9vdujZ0aVCQ/CGBYIkEB0=" via-port "2-1.1" with-interface 08:06:50`

	got, err := ParseLine(line)
	if err != nil {
		t.Fatal(err)
	}

	if got.Device.RuntimeID != "31" {
		t.Fatalf(
			"runtime id = %q, want 31",
			got.Device.RuntimeID,
		)
	}

	if got.Target != "block" {
		t.Fatalf("target = %q, want block", got.Target)
	}

	if got.Device.Identity.Interfaces[0] != "08:06:50" {
		t.Fatalf(
			"interfaces = %#v",
			got.Device.Identity.Interfaces,
		)
	}
}

func TestParseDockCompositePreservesInterfaces(t *testing.T) {
	line := `44: block id 372e:1019 serial "000000000001" name "M3" hash "+MMaLj7a4LfMlDYYWrlwfcJcdEzm1B5xGG8tqZbQQYw=" parent-hash "1H2ky9gtbbS/N82LVsz28SAzkuwV8+1KCETppm9CiXI=" via-port "7-1.1.4" with-interface { 03:01:02 03:01:01 03:01:00 } with-connect-type "unknown"`

	got, err := ParseLine(line)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"03:01:02",
		"03:01:01",
		"03:01:00",
	}

	if !reflect.DeepEqual(
		got.Device.Identity.Interfaces,
		want,
	) {
		t.Fatalf(
			"interfaces = %#v, want %#v",
			got.Device.Identity.Interfaces,
			want,
		)
	}

	if got.Device.Identity.ParentHash == "" {
		t.Fatal("dock child lost parent hash")
	}

	if got.Device.Identity.Port != "7-1.1.4" {
		t.Fatalf(
			"dock child port = %q",
			got.Device.Identity.Port,
		)
	}
}

func TestParseRepeatedInterfacesPreservesIdentity(t *testing.T) {
	line := `allow id 13d3:3571 serial "00e04c000001" name "Bluetooth Radio" hash "ARE3PgxKUmKQRiOYUhdfinteOxSPASNRsO3gkcBs+p8=" parent-hash "p9XdYtshh7dWZvylwnT2mODxf1un62IZMB3s1EZjqbs=" with-interface { e0:01:01 e0:01:01 e0:01:01 } with-connect-type "hardwired"`

	got, err := ParseLine(line)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"e0:01:01",
		"e0:01:01",
		"e0:01:01",
	}

	if !reflect.DeepEqual(
		got.Device.Identity.Interfaces,
		want,
	) {
		t.Fatalf(
			"repeated interfaces = %#v, want %#v",
			got.Device.Identity.Interfaces,
			want,
		)
	}
}

func TestParserFailsClosedOnUnknownAttribute(t *testing.T) {
	line := `allow id 1234:5678 hash "abc" surprise "nope"`

	if _, err := ParseLine(line); err == nil {
		t.Fatal("accepted unknown USBGuard attribute")
	}
}

func TestParserRequiresHash(t *testing.T) {
	line := `block id 1234:5678 with-interface 03:01:01`

	if _, err := ParseLine(line); err == nil {
		t.Fatal("accepted observation without descriptor hash")
	}
}
