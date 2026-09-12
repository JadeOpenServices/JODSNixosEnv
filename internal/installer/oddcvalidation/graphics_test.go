package oddcvalidation

import (
	"context"
	"testing"

	"github.com/bakanura/gjallarOS/internal/hardware/graphics"
)

func TestRuntimeGraphicsGateMatchesGeneratedState(t *testing.T) {
	runner := func(
		ctx context.Context,
		dir string,
		name string,
		args ...string,
	) ([]byte, error) {
		return []byte(`{
		  "vendor":"amd",
		  "deviceId":"15bf",
		  "type":"integrated",
		  "compute":true,
		  "busId":"PCI:193:0:0",
		  "integratedBusId":"PCI:193:0:0"
		}`), nil
	}

	detect := func(context.Context) (graphics.Result, error) {
		return graphics.Result{
			Vendor:          "amd",
			DeviceID:        "15bf",
			Type:            "integrated",
			Compute:         true,
			BusID:           "PCI:193:0:0",
			IntegratedBusID: "PCI:193:0:0",
		}, nil
	}

	result := runtimeGraphicsResult(
		context.Background(),
		runner,
		detect,
		"/repo",
	)

	if !result.Passed {
		t.Fatalf("matching graphics state rejected: %+v", result)
	}
}

func TestRuntimeGraphicsGateRejectsMismatch(t *testing.T) {
	runner := func(
		ctx context.Context,
		dir string,
		name string,
		args ...string,
	) ([]byte, error) {
		return []byte(`{
		  "vendor":"amd",
		  "deviceId":"15bf",
		  "type":"integrated",
		  "compute":true,
		  "busId":"PCI:193:0:0",
		  "integratedBusId":"PCI:193:0:0"
		}`), nil
	}

	detect := func(context.Context) (graphics.Result, error) {
		return graphics.Result{
			Vendor:   "amd",
			DeviceID: "ffff",
			Type:     "integrated",
			Compute:  true,
			BusID:    "PCI:193:0:0",
		}, nil
	}

	result := runtimeGraphicsResult(
		context.Background(),
		runner,
		detect,
		"/repo",
	)

	if result.Passed {
		t.Fatal("mismatched runtime graphics was accepted")
	}
}
