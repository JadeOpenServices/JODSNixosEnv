package main

import "testing"

func TestParseBusctlObjectPath(t *testing.T) {
	path, ok := parseBusctlObjectPath(`o "/net/reactivated/Fprint/Device/0"` + "\n")
	if !ok || path != "/net/reactivated/Fprint/Device/0" {
		t.Fatalf("got %q %t", path, ok)
	}
	for _, bad := range []string{"", `s "x"`, `o "/org/other/0"`} {
		if _, ok := parseBusctlObjectPath(bad); ok {
			t.Fatalf("accepted %q", bad)
		}
	}
}

func TestBusctlArrayLength(t *testing.T) {
	if n := busctlArrayLength(`as 1 "right-index-finger"` + "\n"); n != 1 {
		t.Fatalf("got %d", n)
	}
	for _, empty := range []string{"", "as 0", `o "/x"`, "as x"} {
		if n := busctlArrayLength(empty); n != 0 {
			t.Fatalf("%q gave %d", empty, n)
		}
	}
}
