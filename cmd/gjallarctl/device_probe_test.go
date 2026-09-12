package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bakanura/gjallarOS/internal/hardware/deviceprobe"
)

func TestDeviceProbeDiagnoseRefusesNonZBookWithoutForce(t *testing.T) {
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
	if stdout.Len() != 0 {
		t.Fatalf("device-specific gates ran before refusal: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "refusing device-specific diagnostics") {
		t.Fatalf("missing refusal message: %q", stderr.String())
	}
}

func TestDeviceProbeDiagnoseForceAllowsDevelopmentRun(t *testing.T) {
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
		[]string{"zbook-x2-g4", "--input", path, "--force"},
		&stdout,
		&stderr,
	)

	if rc != 1 {
		t.Fatalf("return code=%d, want 1 because forced identity gate still fails", rc)
	}
	if !strings.Contains(stdout.String(), "FAIL: identity:") {
		t.Fatalf("forced diagnostics did not run: %q", stdout.String())
	}
	if strings.Contains(stderr.String(), "refusing device-specific diagnostics") {
		t.Fatalf("force unexpectedly refused diagnostics: %q", stderr.String())
	}
}
