package sysfssource

import (
	"reflect"
	"testing"

	"github.com/JadeOpenServices/gjallarOS/internal/hardware/deviceprobe"
	"github.com/JadeOpenServices/gjallarOS/internal/usbtrust"
)

func TestObservedPreservesOnlyKnownSysfsIdentity(t *testing.T) {
	snapshot := deviceprobe.Snapshot{
		USBDevices: []deviceprobe.USBDevice{
			{
				Path:    "1-4",
				Vendor:  "27C6",
				Product: "609C",
				Class:   "00",
				Driver:  "usb",
			},
		},
	}

	got := Observed(snapshot)

	want := []usbtrust.ObservedDevice{
		{
			RuntimeID: "1-4",
			Identity: usbtrust.Identity{
				VIDPID: "27c6:609c",
			},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("observations = %#v, want %#v", got, want)
	}

	identity := got[0].Identity
	if identity.Hash != "" ||
		identity.Serial != "" ||
		identity.ParentHash != "" ||
		identity.Port != "" ||
		identity.ConnectType != "" ||
		len(identity.Interfaces) != 0 {
		t.Fatalf("sysfs adapter fabricated identity facts: %+v", identity)
	}
}

func TestObservedFiltersRootHubs(t *testing.T) {
	snapshot := deviceprobe.Snapshot{
		USBDevices: []deviceprobe.USBDevice{
			{
				Path:    "usb1",
				Vendor:  "1d6b",
				Product: "0002",
			},
			{
				Path:    "1-4",
				Vendor:  "1234",
				Product: "5678",
			},
		},
	}

	got := Observed(snapshot)

	if len(got) != 1 || got[0].RuntimeID != "1-4" {
		t.Fatalf("root hub leaked into observations: %#v", got)
	}
}
