package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"
)

const DefaultSocket = "/run/gjallar-usbtrust/control.sock"

func Call(ctx context.Context, socket string, request Request) (Response, error) {
	if err := ValidateRequest(request); err != nil {
		return Response{}, err
	}
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "unix", socket)
	if err != nil {
		return Response{}, err
	}
	defer conn.Close()
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(30 * time.Second)
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return Response{}, err
	}
	if err := json.NewEncoder(conn).Encode(request); err != nil {
		return Response{}, err
	}
	if err := conn.(*net.UnixConn).CloseWrite(); err != nil {
		return Response{}, err
	}
	var response Response
	if err := json.NewDecoder(io.LimitReader(conn, 4*1024*1024)).Decode(&response); err != nil {
		return response, err
	}
	if !response.OK {
		return response, fmt.Errorf("%s", response.Error)
	}
	return response, nil
}
