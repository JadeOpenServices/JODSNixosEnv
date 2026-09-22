package broker

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/bakanura/gjallarOS/internal/usbtrust"
)

type fakeReader struct {
	status      Status
	statusErr   error
	audit       usbtrust.AuditResult
	revision    uint64
	auditErr    error
	statusCalls int
	auditCalls  int
}

func (f *fakeReader) Status(
	context.Context,
) (Status, error) {
	f.statusCalls++
	return f.status, f.statusErr
}

func (f *fakeReader) Audit(
	context.Context,
) (usbtrust.AuditResult, uint64, error) {
	f.auditCalls++
	return f.audit, f.revision, f.auditErr
}

func TestHandlerOwnerStatus(t *testing.T) {
	reader := &fakeReader{
		status: Status{
			StatePresent:    true,
			Revision:        7,
			TrustedDevices:  2,
			ExpectedDevices: 2,
			ObservedDevices: 5,
		},
	}

	handler := Handler{
		OwnerUID: 1000,
		Reader:   reader,
	}

	response := handler.Handle(
		context.Background(),
		1000,
		Request{
			Action: ActionStatus,
		},
	)

	if !response.OK {
		t.Fatalf(
			"status failed: %s",
			response.Error,
		)
	}

	if response.Status == nil {
		t.Fatal("status response missing status payload")
	}

	if response.Status.Revision != 7 {
		t.Fatalf(
			"revision = %d, want 7",
			response.Status.Revision,
		)
	}

	if reader.statusCalls != 1 {
		t.Fatalf(
			"status calls = %d, want 1",
			reader.statusCalls,
		)
	}
}

func TestHandlerOwnerAudit(t *testing.T) {
	reader := &fakeReader{
		revision: 9,
		audit: usbtrust.AuditResult{
			Findings: []usbtrust.Finding{
				{
					State:   usbtrust.AuditBlock,
					Code:    usbtrust.CodeUnknownExternal,
					Message: "synthetic",
				},
			},
		},
	}

	handler := Handler{
		OwnerUID: 1000,
		Reader:   reader,
	}

	response := handler.Handle(
		context.Background(),
		1000,
		Request{
			Action: ActionAudit,
		},
	)

	if !response.OK {
		t.Fatalf(
			"audit failed: %s",
			response.Error,
		)
	}

	if response.Audit == nil {
		t.Fatal("audit response missing audit payload")
	}

	if response.Revision != 9 {
		t.Fatalf(
			"revision = %d, want 9",
			response.Revision,
		)
	}
}

func TestHandlerRootMutationStillFailsClosed(t *testing.T) {
	reader := &fakeReader{}

	handler := Handler{
		OwnerUID: 1000,
		Reader:   reader,
	}

	portable := true

	response := handler.Handle(
		context.Background(),
		0,
		Request{
			Action:     ActionTrustPermanent,
			RuntimeID:  "7",
			Connection: "review-token",
			Portable:   &portable,
		},
	)

	if response.OK {
		t.Fatal("root mutation unexpectedly enabled")
	}

	if !strings.Contains(
		response.Error,
		"is not enabled",
	) {
		t.Fatalf(
			"unexpected mutation error %q",
			response.Error,
		)
	}

	if reader.statusCalls != 0 ||
		reader.auditCalls != 0 {
		t.Fatal(
			"mutation request reached read model",
		)
	}
}

func TestHandlerRejectsUnrelatedUIDBeforeReader(
	t *testing.T,
) {
	reader := &fakeReader{}

	handler := Handler{
		OwnerUID: 1000,
		Reader:   reader,
	}

	response := handler.Handle(
		context.Background(),
		2000,
		Request{
			Action: ActionStatus,
		},
	)

	if response.OK {
		t.Fatal("unrelated UID received status")
	}

	if reader.statusCalls != 0 {
		t.Fatal(
			"unauthorized request reached read model",
		)
	}
}

func TestHandlerPropagatesReaderFailure(t *testing.T) {
	reader := &fakeReader{
		statusErr: errors.New("synthetic failure"),
	}

	handler := Handler{
		OwnerUID: 1000,
		Reader:   reader,
	}

	response := handler.Handle(
		context.Background(),
		1000,
		Request{
			Action: ActionStatus,
		},
	)

	if response.OK {
		t.Fatal("reader failure became success")
	}

	if !strings.Contains(
		response.Error,
		"synthetic failure",
	) {
		t.Fatalf(
			"unexpected error %q",
			response.Error,
		)
	}
}

func TestReadBoundedRequest(t *testing.T) {
	request, err := readBoundedRequest(
		strings.NewReader(
			`{"action":"audit"}`,
		),
	)
	if err != nil {
		t.Fatal(err)
	}

	if request.Action != ActionAudit {
		t.Fatalf(
			"action = %q, want audit",
			request.Action,
		)
	}
}

func TestReadBoundedRequestRejectsOversize(
	t *testing.T,
) {
	data := bytes.Repeat(
		[]byte{'x'},
		MaxRequestBytes+1,
	)

	if _, err := readBoundedRequest(
		bytes.NewReader(data),
	); err == nil {
		t.Fatal("accepted oversized broker request")
	}
}

func TestReadBoundedRequestRejectsEmpty(t *testing.T) {
	if _, err := readBoundedRequest(
		strings.NewReader(" \n\t"),
	); err == nil {
		t.Fatal("accepted empty broker request")
	}
}
