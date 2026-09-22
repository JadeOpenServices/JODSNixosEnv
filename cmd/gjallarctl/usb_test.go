package main

import (
	"bytes"
	"encoding/json"
	"net"
	"path/filepath"
	"testing"

	"github.com/bakanura/gjallarOS/internal/usbtrust/broker"
)

func TestUSBCLIUsesBrokerAndReportsFailures(t *testing.T) {
	for _, success := range []bool{true, false} {
		t.Run(map[bool]string{true: "success", false: "rejected"}[success], func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "broker.sock")
			listener, err := net.Listen("unix", path)
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			done := make(chan broker.Request, 1)
			go func() {
				conn, err := listener.Accept()
				if err != nil {
					return
				}
				defer conn.Close()
				request, err := broker.DecodeRequest(conn)
				if err != nil {
					return
				}
				done <- request
				_ = json.NewEncoder(conn).Encode(broker.Response{OK: success, Error: "synthetic rejection"})
			}()
			var out, errors bytes.Buffer
			code := runUSB([]string{"allow-once", "--runtime-id", "12", "--connection", "review-token", "--socket", path}, &out, &errors)
			if (code == 0) != success {
				t.Fatalf("code %d: %s", code, errors.String())
			}
			request := <-done
			if request.Action != broker.ActionAllowOnce || request.Connection != "review-token" {
				t.Fatal(request)
			}
		})
	}
}

func TestUSBCLIPermanentRequiresPortability(t *testing.T) {
	var out, err bytes.Buffer
	if code := runUSB([]string{"trust-permanent", "--runtime-id", "12"}, &out, &err); code != 2 {
		t.Fatalf("code %d", code)
	}
}
