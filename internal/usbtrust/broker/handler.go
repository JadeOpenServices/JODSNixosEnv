package broker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"

	"github.com/bakanura/gjallarOS/internal/usbtrust"
)

const MaxRequestBytes = 16 * 1024

type Reader interface {
	Status(context.Context) (Status, error)
	Audit(context.Context) (usbtrust.AuditResult, uint64, error)
}

type Handler struct {
	OwnerUID uint32
	Reader   Reader
}

func (h Handler) Handle(
	ctx context.Context,
	peerUID uint32,
	request Request,
) Response {
	if err := ValidateRequest(request); err != nil {
		return errorResponse(err)
	}

	if err := AuthorizePeer(
		peerUID,
		request.Action,
		h.OwnerUID,
	); err != nil {
		return errorResponse(err)
	}

	// Even root receives no mutation path until the mutation engine,
	// interactive authorization and signed-state transaction are wired.
	if request.Action.Mutation() {
		return errorResponse(fmt.Errorf(
			"USB trust mutation %q is not enabled",
			request.Action,
		))
	}

	if h.Reader == nil {
		return errorResponse(fmt.Errorf(
			"USB trust read model is unavailable",
		))
	}

	switch request.Action {
	case ActionStatus:
		status, err := h.Reader.Status(ctx)
		if err != nil {
			return errorResponse(fmt.Errorf(
				"read USB trust status: %w",
				err,
			))
		}

		return Response{
			OK:       true,
			Revision: status.Revision,
			Status:   &status,
		}

	case ActionAudit:
		result, revision, err := h.Reader.Audit(ctx)
		if err != nil {
			return errorResponse(fmt.Errorf(
				"audit USB trust state: %w",
				err,
			))
		}

		return Response{
			OK:       true,
			Revision: revision,
			Audit:    &result,
		}

	default:
		return errorResponse(fmt.Errorf(
			"USB trust action %q has no handler",
			request.Action,
		))
	}
}

func (h Handler) ServeConn(
	ctx context.Context,
	conn *net.UnixConn,
) error {
	if conn == nil {
		return fmt.Errorf("USB trust connection is nil")
	}

	peerUID, err := PeerUID(conn)
	if err != nil {
		return err
	}

	request, err := readBoundedRequest(conn)
	if err != nil {
		return EncodeResponse(
			conn,
			errorResponse(err),
		)
	}

	return EncodeResponse(
		conn,
		h.Handle(ctx, peerUID, request),
	)
}

func readBoundedRequest(
	reader io.Reader,
) (Request, error) {
	limited := &io.LimitedReader{
		R: reader,
		N: MaxRequestBytes + 1,
	}

	data, err := io.ReadAll(limited)
	if err != nil {
		return Request{}, fmt.Errorf(
			"read USB trust request: %w",
			err,
		)
	}

	if len(data) > MaxRequestBytes {
		return Request{}, fmt.Errorf(
			"USB trust request exceeds %d bytes",
			MaxRequestBytes,
		)
	}

	if len(bytes.TrimSpace(data)) == 0 {
		return Request{}, fmt.Errorf(
			"USB trust request is empty",
		)
	}

	return DecodeRequest(bytes.NewReader(data))
}

func errorResponse(err error) Response {
	return Response{
		OK:    false,
		Error: err.Error(),
	}
}
