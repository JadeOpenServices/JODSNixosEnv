// Package controller composes domain decisions, verified persistence and live
// observations. It is used only by the privileged USB trust daemon.
package controller

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/bakanura/gjallarOS/internal/usbtrust"
	"github.com/bakanura/gjallarOS/internal/usbtrust/broker"
	"github.com/bakanura/gjallarOS/internal/usbtrust/oddcsource"
	"github.com/bakanura/gjallarOS/internal/usbtrust/readmodel"
)

type Controller struct {
	Reader                       readmodel.Reader
	MachineID, ModelID, StateDir string
	Signer                       usbtrust.Signer
	// Apply receives derived policy only; no adapter may create trust records.
	Apply            func(context.Context, []usbtrust.Decision) error
	Commit           func(context.Context, usbtrust.Document) error
	Provision        func(context.Context) error
	mu               sync.Mutex
	sessions         usbtrust.Sessions
	persistenceReady bool
}

func (c *Controller) EnforcementEnabled() bool { return c.Apply != nil }

func (c *Controller) SetPersistenceReady(ready bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.persistenceReady = ready
}

func (c *Controller) Status(ctx context.Context) (broker.Status, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	input, err := c.snapshot(ctx)
	if err != nil {
		return broker.Status{}, err
	}
	s := broker.Status{
		StatePresent:        input.Trusted != nil,
		PermanentTrustReady: c.persistenceReady,
		ExpectedDevices:     len(input.Expected),
		ObservedDevices:     len(input.Observed),
	}
	s.Enforcing = c.EnforcementEnabled()
	if input.Trusted != nil {
		s.Revision = input.Trusted.Revision
		s.TrustedDevices = len(input.Trusted.Devices)
		s.Devices = append([]usbtrust.Device(nil), input.Trusted.Devices...)
	}
	return s, nil
}

func (c *Controller) Audit(ctx context.Context) (usbtrust.AuditResult, uint64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	input, err := c.snapshot(ctx)
	if err != nil {
		return usbtrust.AuditResult{}, 0, err
	}
	result, err := usbtrust.Audit(input)
	return result, revision(input), err
}

func (c *Controller) Policy(ctx context.Context) ([]usbtrust.Decision, uint64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	input, err := c.snapshot(ctx)
	if err != nil {
		return nil, 0, err
	}
	policy, err := c.sessions.Derive(input)
	if err == nil && !c.persistenceReady {
		for i := range policy {
			policy[i].Actions = transientActionsOnly(policy[i].Actions)
		}
	}
	return policy, revision(input), err
}

func (c *Controller) Reconcile(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	input, err := c.snapshot(ctx)
	if err != nil {
		return err
	}
	return c.apply(ctx, input)
}

func (c *Controller) snapshot(ctx context.Context) (input usbtrust.AuditInput, err error) {
	defer func() {
		if err != nil {
			c.sessions.Reset()
		}
	}()
	if c.Reader.Resolved == nil || c.Reader.Trust == nil || c.Reader.Observations == nil {
		return input, fmt.Errorf("USB trust sources are incomplete")
	}
	resolved, err := c.Reader.Resolved.Resolved(ctx)
	if err != nil {
		return input, err
	}
	input.Expected, err = oddcsource.Expected(resolved)
	if err != nil {
		return input, err
	}
	input.Trusted, err = c.Reader.Trust.Trusted(ctx)
	if err != nil {
		return input, err
	}
	if input.Trusted != nil {
		if err := input.Trusted.Validate(); err != nil {
			return input, err
		}
		if input.Trusted.MachineID != c.MachineID || input.Trusted.ODDCModel != c.ModelID {
			return input, fmt.Errorf("USB trust state belongs to a different machine or ODDC model")
		}
	}
	input.Observed, err = c.Reader.Observations.Observed(ctx)
	if err != nil {
		return input, err
	}
	err = c.sessions.Observe(input.Observed)
	return input, err
}

func (c *Controller) Mutate(ctx context.Context, request broker.Request) (broker.Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := broker.ValidateRequest(request); err != nil {
		return broker.Response{}, err
	}
	if request.Action == broker.ActionProvisionKey {
		if c.Provision == nil {
			return broker.Response{}, fmt.Errorf("TPM provisioning is not configured")
		}
		if err := c.Provision(ctx); err != nil {
			return broker.Response{}, err
		}
		c.persistenceReady = true
		return broker.Response{OK: true, Message: "TPM signing key provisioned and verified", Enforcing: c.EnforcementEnabled()}, nil
	}
	if persistentMutation(request.Action) && !c.persistenceReady {
		return broker.Response{}, fmt.Errorf("permanent USB trust is unavailable until the TPM signing key is provisioned and verified")
	}
	input, err := c.snapshot(ctx)
	if err != nil {
		return broker.Response{}, err
	}
	if request.Action != broker.ActionForget {
		if err := c.sessions.Check(request.RuntimeID, request.Connection); err != nil {
			return broker.Response{}, err
		}
	}
	if request.Action == broker.ActionAllowOnce || request.Action == broker.ActionKeepBlocked {
		target := usbtrust.TargetAllow
		if request.Action == broker.ActionKeepBlocked {
			target = usbtrust.TargetBlock
		}
		if err := c.sessions.Choose(request.RuntimeID, request.Connection, target); err != nil {
			return broker.Response{}, err
		}
	} else {
		previousPolicy, err := c.sessions.Derive(input)
		if err != nil {
			return broker.Response{}, err
		}
		document, err := c.change(input, request)
		if err != nil {
			return broker.Response{}, err
		}
		commit := c.Commit
		if commit == nil {
			commit = func(ctx context.Context, doc usbtrust.Document) error {
				return usbtrust.CommitSigned(ctx, c.StateDir, doc, c.Signer)
			}
		}
		if err := commit(ctx, document); err != nil {
			return broker.Response{}, fmt.Errorf("commit USB trust: %w", err)
		}
		input.Trusted = &document
		// Revoke choices/reviews for affected identities only. Trusting a disk
		// must not discard an unrelated keyboard's current allow-once grant.
		for _, decision := range previousPolicy {
			if decision.RuntimeID == request.RuntimeID || (request.TrustedID != "" && decision.TrustedID == request.TrustedID) {
				c.sessions.Disconnect(decision.RuntimeID)
			}
		}
		if err := c.sessions.Observe(input.Observed); err != nil {
			return broker.Response{}, err
		}
	}
	if err := c.apply(ctx, input); err != nil {
		return broker.Response{}, fmt.Errorf("decision recorded; enforcement failed: %w", err)
	}
	return broker.Response{OK: true, Revision: revision(input), Message: "USB trust decision recorded", Enforcing: c.EnforcementEnabled()}, nil
}

func persistentMutation(action broker.Action) bool {
	switch action {
	case broker.ActionTrustPermanent,
		broker.ActionForget,
		broker.ActionAcceptReplacement,
		broker.ActionEnrollInternal:
		return true
	default:
		return false
	}
}

func transientActionsOnly(actions []usbtrust.Action) []usbtrust.Action {
	result := make([]usbtrust.Action, 0, len(actions))
	for _, action := range actions {
		switch action {
		case usbtrust.ActionKeepBlocked, usbtrust.ActionAllowOnce:
			result = append(result, action)
		}
	}
	return result
}

func (c *Controller) apply(ctx context.Context, input usbtrust.AuditInput) error {
	policy, err := c.sessions.Derive(input)
	if err != nil {
		return err
	}
	if c.Apply != nil {
		return c.Apply(ctx, policy)
	}
	return nil
}

func (c *Controller) change(input usbtrust.AuditInput, request broker.Request) (usbtrust.Document, error) {
	doc := usbtrust.Document{Schema: usbtrust.SchemaVersion, MachineID: c.MachineID, ODDCModel: c.ModelID}
	if input.Trusted != nil {
		doc = *input.Trusted
		doc.Devices = append([]usbtrust.Device(nil), doc.Devices...)
	}
	if doc.Revision == ^uint64(0) {
		return doc, fmt.Errorf("USB trust revision exhausted")
	}
	doc.Revision++
	index := -1
	for i, device := range doc.Devices {
		if device.ID == request.TrustedID {
			index = i
		}
	}
	if request.Action == broker.ActionForget {
		if index < 0 {
			return doc, fmt.Errorf("trusted device not found")
		}
		doc.Devices = append(doc.Devices[:index], doc.Devices[index+1:]...)
		return doc, doc.Validate()
	}
	var observed usbtrust.ObservedDevice
	for _, device := range input.Observed {
		if device.RuntimeID == request.RuntimeID {
			observed = device
		}
	}
	policy, err := c.sessions.Derive(input)
	if err != nil {
		return doc, err
	}
	var decision usbtrust.Decision
	for _, d := range policy {
		if d.RuntimeID == request.RuntimeID {
			decision = d
		}
	}
	// Enrollment/replacement validation uses persistent classification, not a
	// temporary allow/block choice that changes the presented policy reason.
	audit, err := usbtrust.Audit(input)
	if err != nil {
		return doc, err
	}
	for _, finding := range audit.Findings {
		if finding.RuntimeID == request.RuntimeID {
			decision.Reason = string(finding.Code)
		}
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return doc, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	device := usbtrust.Device{ID: "device:" + hex.EncodeToString(id[:]), Class: usbtrust.ClassExternal, Identity: observed.Identity, FirstAccepted: now, LastAccepted: now, Strength: usbtrust.StrengthDescriptor}
	if device.Identity.Serial != "" {
		device.Strength = usbtrust.StrengthSerialDescriptor
	}
	switch request.Action {
	case broker.ActionTrustPermanent:
		if decision.Role != "" || decision.TrustedID != "" || observed.Identity.ConnectType == "hardwired" {
			return doc, fmt.Errorf("use internal enrollment or explicit replacement for this device")
		}
		device.Portable = *request.Portable
	case broker.ActionEnrollInternal:
		if decision.Role != request.Role || decision.Reason != string(usbtrust.CodeInternalUnvalidated) {
			return doc, fmt.Errorf("device is not an unambiguous unvalidated ODDC expectation")
		}
		device.Class, device.Role, device.ExpectedByODDC = usbtrust.ClassInternal, request.Role, true
	case broker.ActionAcceptReplacement:
		if index < 0 {
			return doc, fmt.Errorf("trusted device not found")
		}
		old := doc.Devices[index]
		if old.Class == usbtrust.ClassInternal && (decision.Role != old.Role || decision.TrustedID != old.ID || decision.Reason != string(usbtrust.CodeInternalChanged)) {
			return doc, fmt.Errorf("replacement must be an unambiguous changed identity for the accepted ODDC role")
		}
		if old.Class == usbtrust.ClassExternal && (decision.Role != "" || observed.Identity.ConnectType == "hardwired") {
			return doc, fmt.Errorf("external replacement cannot enroll internal hardware")
		}
		device.ID, device.Class, device.Role, device.Portable, device.ExpectedByODDC, device.FirstAccepted = old.ID, old.Class, old.Role, old.Portable, old.ExpectedByODDC, old.FirstAccepted
	default:
		return doc, fmt.Errorf("unsupported USB trust mutation %q", request.Action)
	}
	if !device.Portable && device.Identity.ParentHash == "" && device.Identity.Port == "" {
		return doc, fmt.Errorf("non-portable trust requires observed topology")
	}
	if !device.Portable && device.Identity.Serial != "" {
		device.Strength = usbtrust.StrengthSerialDescriptorTopology
	}
	if index >= 0 {
		doc.Devices[index] = device
	} else {
		doc.Devices = append(doc.Devices, device)
	}
	return doc, doc.Validate()
}

func revision(input usbtrust.AuditInput) uint64 {
	if input.Trusted != nil {
		return input.Trusted.Revision
	}
	return 0
}
