package main

import (
	"context"
	"errors"
	"testing"

	"github.com/godbus/dbus/v5"
)

type fakeFprintBus struct {
	activatable []string
	uid         uint32
	deviceErr   error
	fingers     []string
	fingersErr  error
}

func (f fakeFprintBus) Activatable(context.Context) ([]string, error) {
	return f.activatable, nil
}

func (f fakeFprintBus) OwnerUID(context.Context, string) (uint32, error) {
	return f.uid, nil
}

func (f fakeFprintBus) DefaultDevice(context.Context) (dbus.ObjectPath, error) {
	return "/net/reactivated/Fprint/Device/0", f.deviceErr
}

func (f fakeFprintBus) EnrolledFingers(context.Context, dbus.ObjectPath) ([]string, error) {
	return f.fingers, f.fingersErr
}

func TestProbeFingerprintStates(t *testing.T) {
	present := []string{"org.freedesktop.systemd1", fprintName}
	cases := []struct {
		name string
		bus  fakeFprintBus
		want fingerprintState
	}{
		{"enrolled", fakeFprintBus{activatable: present, fingers: []string{"right-index-finger"}}, fingerprintReady},
		{"no fprintd", fakeFprintBus{activatable: []string{"org.freedesktop.systemd1"}}, fingerprintNoService},
		{"no reader", fakeFprintBus{activatable: present,
			deviceErr: dbus.Error{Name: fprintName + ".Error.NoSuchDevice"}}, fingerprintNoReader},
		{"no prints error", fakeFprintBus{activatable: present,
			fingersErr: dbus.Error{Name: fprintName + ".Error.NoEnrolledPrints"}}, fingerprintNoPrints},
		{"no prints empty", fakeFprintBus{activatable: present}, fingerprintNoPrints},
		{"not root", fakeFprintBus{activatable: present, uid: 1000,
			fingers: []string{"right-index-finger"}}, fingerprintUntrusted},
		{"other error", fakeFprintBus{activatable: present,
			fingersErr: errors.New("timeout")}, fingerprintProbeFailed},
	}
	for _, c := range cases {
		got := probeFingerprintOn(context.Background(), c.bus)
		if got.State != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got.State, c.want)
		}
		if c.want != fingerprintReady && got.Message() == "" {
			t.Errorf("%s: no message", c.name)
		}
	}
}
