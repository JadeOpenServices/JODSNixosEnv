package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/bakanura/gjallarOS/internal/installer/oddcvalidation"
)

func TestODDCValidateDeviceCLIPrintsSuccessfulGate(t *testing.T) {
	var stdout, stderr bytes.Buffer

	run := func(
		context.Context,
		string,
		oddcvalidation.CommandRunner,
	) (oddcvalidation.Report, oddcvalidation.DeviceContext, error) {
		return oddcvalidation.Report{
			Results: []oddcvalidation.Result{
				{
					Gate:   oddcvalidation.GateGoTests,
					Passed: true,
				},
			},
		}, oddcvalidation.DeviceContext{}, nil
	}

	rc := runODDCValidateDeviceWith(
		"/repo",
		&stdout,
		&stderr,
		run,
	)

	if rc != 0 {
		t.Fatalf("exit=%d stderr=%q", rc, stderr.String())
	}
	if !strings.Contains(stdout.String(), "PASS: go-tests") {
		t.Fatalf("missing gate output: %q", stdout.String())
	}
}

func TestODDCValidateDeviceCLIReturnsFailure(t *testing.T) {
	var stdout, stderr bytes.Buffer

	run := func(
		context.Context,
		string,
		oddcvalidation.CommandRunner,
	) (oddcvalidation.Report, oddcvalidation.DeviceContext, error) {
		return oddcvalidation.Report{
			Results: []oddcvalidation.Result{
				{
					Gate:    oddcvalidation.GateInstallerCheck,
					Details: "project WARN present",
				},
			},
		}, oddcvalidation.DeviceContext{}, errors.New("validation failed")
	}

	rc := runODDCValidateDeviceWith(
		"/repo",
		&stdout,
		&stderr,
		run,
	)

	if rc != 1 {
		t.Fatalf("exit=%d", rc)
	}
	if !strings.Contains(stdout.String(), "FAIL: installer-check") {
		t.Fatalf("missing failed gate: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "ERROR: validation failed") {
		t.Fatalf("missing error: %q", stderr.String())
	}
}
