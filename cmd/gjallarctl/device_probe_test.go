package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bakanura/gjallarOS/internal/hardware/deviceprobe"
)

func TestDeviceProbeDiagnoseWrongIdentityRunsIdentityGate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "snapshot.json")

	if err := deviceprobe.WriteSnapshot(path, deviceprobe.Snapshot{
		Schema:      1,
		SysVendor:   "Framework",
		ProductName: "Laptop 13 (AMD Ryzen 7040Series)",
		BoardName:   "FRANMDCP07",
	}); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	rc := runDeviceProbeDiagnose(
		[]string{"zbook-x2-g4", "--input", path},
		&stdout,
		&stderr,
	)

	if rc != 1 {
		t.Fatalf("return code=%d, want 1", rc)
	}
	if !strings.Contains(stdout.String(), "FAIL: identity:") {
		t.Fatalf("identity gate did not reject wrong device: %q", stdout.String())
	}
	if strings.Contains(stderr.String(), "refusing device-specific diagnostics") {
		t.Fatalf("legacy CLI refusal remains: %q", stderr.String())
	}
}
