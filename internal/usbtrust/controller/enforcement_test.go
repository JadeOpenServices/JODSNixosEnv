package controller

import (
	"context"
	"errors"
	"testing"

	"github.com/bakanura/gjallarOS/internal/usbtrust"
	"github.com/bakanura/gjallarOS/internal/usbtrust/broker"
)

func armedFixture(t *testing.T) (*Controller, *sources, *[]usbtrust.Target) {
	t.Helper()
	c, s := fixture()
	s.resolved = map[string]any{"hardware": map[string]any{"input": map[string]any{"bus": "usb", "attachment": "internal", "deviceId": "1234:5678"}}}
	s.observed = append(s.observed, usbtrust.ObservedDevice{RuntimeID: "13", Identity: usbtrust.Identity{VIDPID: "dead:beef", Hash: "stick", Port: "1-3", ConnectType: "hotplug", Interfaces: []string{"08:06:50"}}})
	var implicit []usbtrust.Target
	c.ArmOnStart = true
	c.Apply = func(context.Context, []usbtrust.Decision) error { return nil }
	c.SetImplicitTarget = func(_ context.Context, target usbtrust.Target) error {
		implicit = append(implicit, target)
		return nil
	}
	c.VerifyRecoveryKey = func(_ context.Context, key []byte) error {
		if string(key) != "correct horse" {
			return errors.New("no key slot")
		}
		return nil
	}
	if err := c.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	return c, s, &implicit
}

func TestFirstArmingEnrollsInternalDevicesOnly(t *testing.T) {
	c, s, implicit := armedFixture(t)
	if s.doc == nil || s.doc.Enforcement == nil || !s.doc.Enforcement.Armed {
		t.Fatalf("enforcement not armed: %+v", s.doc)
	}
	if len(s.doc.Devices) != 1 || s.doc.Devices[0].Role != "hardware.input" || s.doc.Devices[0].Class != usbtrust.ClassInternal {
		t.Fatalf("unexpected enrollment: %+v", s.doc.Devices)
	}
	if !c.EnforcementEnabled() || len(*implicit) != 1 || (*implicit)[0] != usbtrust.TargetBlock {
		t.Fatalf("USBGuard fallback not blocking: %v", *implicit)
	}
	policy, _, _ := c.Policy(context.Background())
	for _, d := range policy {
		if d.RuntimeID == "13" && d.Target != usbtrust.TargetBlock {
			t.Fatal("external device allowed by arming")
		}
	}
}

func TestRearmingNeverEnrollsAgain(t *testing.T) {
	c, s, _ := armedFixture(t)
	ctx := context.Background()
	if _, err := c.Mutate(ctx, broker.Request{Action: broker.ActionDisarm, RecoveryKey: "correct horse"}); err != nil {
		t.Fatal(err)
	}
	s.doc.Devices = nil
	// A restarted daemon re-arms but must not trust whatever is attached now.
	restarted := &Controller{Reader: c.Reader, MachineID: c.MachineID, ModelID: c.ModelID, Commit: c.Commit, ArmOnStart: true, Apply: c.Apply}
	restarted.SetPersistenceReady(true)
	if err := restarted.Reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	if !s.doc.Enforcement.Armed || len(s.doc.Devices) != 0 {
		t.Fatalf("re-arming changed trust: %+v", s.doc)
	}
}

func TestDisarmRequiresDiskPassphrase(t *testing.T) {
	c, s, implicit := armedFixture(t)
	ctx := context.Background()
	revision := s.doc.Revision
	if _, err := c.Mutate(ctx, broker.Request{Action: broker.ActionDisarm, RecoveryKey: "guess"}); err == nil {
		t.Fatal("wrong passphrase disarmed enforcement")
	}
	if !s.doc.Enforcement.Armed || s.doc.Revision != revision || !c.EnforcementEnabled() {
		t.Fatal("rejected passphrase changed state")
	}
	if _, err := c.Mutate(ctx, broker.Request{Action: broker.ActionDisarm, RecoveryKey: "correct horse"}); err != nil {
		t.Fatal(err)
	}
	if s.doc.Enforcement.Armed || c.EnforcementEnabled() {
		t.Fatal("disarm did not stop enforcement")
	}
	if last := (*implicit)[len(*implicit)-1]; last != usbtrust.TargetAllow {
		t.Fatalf("USBGuard fallback still %q", last)
	}
	// The running daemon must not re-arm itself right after a disarm.
	if err := c.Reconcile(ctx); err != nil || s.doc.Enforcement.Armed {
		t.Fatalf("re-armed without restart: %v", err)
	}
}

func TestArmedStateEnforcesWithoutFlag(t *testing.T) {
	c, _, _ := armedFixture(t)
	restarted := &Controller{Reader: c.Reader, MachineID: c.MachineID, ModelID: c.ModelID, Apply: c.Apply}
	if err := restarted.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !restarted.EnforcementEnabled() {
		t.Fatal("dropping the enforce flag disabled a signed armed state")
	}
}

func TestDisarmRejectsExtraArguments(t *testing.T) {
	if err := broker.ValidateRequest(broker.Request{Action: broker.ActionDisarm}); err == nil {
		t.Fatal("disarm accepted without passphrase")
	}
	if err := broker.ValidateRequest(broker.Request{Action: broker.ActionAllowOnce, RuntimeID: "1", Connection: "c", RecoveryKey: "x"}); err == nil {
		t.Fatal("passphrase accepted by unrelated action")
	}
}
