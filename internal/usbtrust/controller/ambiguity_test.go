package controller

import (
	"context"
	"testing"

	"github.com/bakanura/gjallarOS/internal/usbtrust"
	"github.com/bakanura/gjallarOS/internal/usbtrust/broker"
)

func TestEnrolledInternalReplacementRequiresUnambiguousCandidate(t *testing.T) {
	c, s := fixture()
	s.resolved = map[string]any{"hardware": map[string]any{"input": map[string]any{
		"bus": "usb", "attachment": "internal", "deviceId": "1234:5678",
	}}}
	ctx := context.Background()
	const role = "hardware.input"
	policy, _, err := c.Policy(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Mutate(ctx, broker.Request{
		Action: broker.ActionEnrollInternal, RuntimeID: "12",
		Connection: policy[0].Connection, Role: role,
	}); err != nil {
		t.Fatal(err)
	}
	accepted := s.doc.Devices[0]
	revision := s.doc.Revision

	s.observed[0].Identity.Hash = "replacement-a"
	duplicate := s.observed[0]
	duplicate.RuntimeID = "13"
	duplicate.Identity.Hash = "replacement-b"
	duplicate.Identity.Port = "1-3"
	s.observed = append(s.observed, duplicate)
	policy, _, err = c.Policy(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(policy) != 2 {
		t.Fatalf("expected two replacement candidates, got %v", policy)
	}
	portable := true
	for _, decision := range policy {
		if decision.Reason != string(usbtrust.CodeInternalAmbiguous) ||
			decision.Role != role || decision.TrustedID != accepted.ID ||
			decision.Target != usbtrust.TargetBlock {
			t.Fatalf("ambiguous candidate lost internal classification: %+v", decision)
		}
		for _, request := range []broker.Request{
			{
				Action: broker.ActionTrustPermanent, RuntimeID: decision.RuntimeID,
				Connection: decision.Connection, Portable: &portable,
			},
			{
				Action: broker.ActionAcceptReplacement, RuntimeID: decision.RuntimeID,
				Connection: decision.Connection, TrustedID: accepted.ID,
			},
		} {
			if _, err := c.Mutate(ctx, request); err == nil {
				t.Fatalf("ambiguous candidate accepted through %s: %+v", request.Action, decision)
			}
		}
	}
	if s.doc.Revision != revision || len(s.doc.Devices) != 1 ||
		s.doc.Devices[0].Identity.Hash != accepted.Identity.Hash {
		t.Fatalf("rejected mutations changed accepted trust: %+v", s.doc)
	}

	s.observed = s.observed[:1]
	policy, _, err = c.Policy(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(policy) != 1 || policy[0].Reason != string(usbtrust.CodeInternalChanged) {
		t.Fatalf("remaining candidate is not a reviewable replacement: %+v", policy)
	}
	if _, err := c.Mutate(ctx, broker.Request{
		Action: broker.ActionAcceptReplacement, RuntimeID: policy[0].RuntimeID,
		Connection: policy[0].Connection, TrustedID: accepted.ID,
	}); err != nil {
		t.Fatal(err)
	}
	if s.doc.Revision != revision+1 || len(s.doc.Devices) != 1 ||
		s.doc.Devices[0].ID != accepted.ID ||
		s.doc.Devices[0].Class != usbtrust.ClassInternal ||
		s.doc.Devices[0].Identity.Hash != "replacement-a" {
		t.Fatalf("unambiguous replacement was not committed: %+v", s.doc)
	}
	policy, _, err = c.Policy(ctx)
	if err != nil || len(policy) != 1 || policy[0].Target != usbtrust.TargetAllow ||
		policy[0].Reason != string(usbtrust.CodeInternalMatch) {
		t.Fatalf("accepted replacement is not allowed: policy=%+v err=%v", policy, err)
	}
}
