package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/bakanura/gjallarOS/internal/installercheck"
)

func TestRebuildDoesNotRequireExplicitRepoAndHost(t *testing.T) {
	t.Setenv("GJALLAROS_REPO", "")
	t.Chdir(t.TempDir())

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	status := runRebuild(nil, &stdout, &stderr)

	if status != 2 {
		t.Fatalf(
			"runRebuild status = %d, want 2 for unresolved repository",
			status,
		)
	}

	if strings.Contains(
		stderr.String(),
		"rebuild requires --repo and --host",
	) {
		t.Fatalf(
			"rebuild still requires explicit repo/host:\n%s",
			stderr.String(),
		)
	}

	if !strings.Contains(
		stderr.String(),
		"resolve GjallarOS repository",
	) {
		t.Fatalf(
			"rebuild did not reach repository resolution:\n%s",
			stderr.String(),
		)
	}
}

func TestRebuildPreflightSuccessIsCompact(t *testing.T) {
	findings := []installercheck.Finding{
		{
			Level:   "PASS",
			Message: "default preset is valid",
		},
		{
			Level:   installercheck.Warn,
			Message: "known optional warning",
		},
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	status := reportRebuildPreflightResult(
		findings,
		false,
		&stdout,
		&stderr,
	)

	if status != 0 {
		t.Fatalf("status = %d, want 0", status)
	}

	got := stdout.String()

	if !strings.Contains(got, "[GjallarOS] Preflight ✓") {
		t.Fatalf("compact success line missing:\n%s", got)
	}

	if !strings.Contains(got, "WARN: known optional warning") {
		t.Fatalf("warning was hidden:\n%s", got)
	}

	if strings.Contains(got, "PASS: default preset is valid") {
		t.Fatalf("successful finding leaked into rebuild output:\n%s", got)
	}

	if strings.Contains(got, "Fast preflight") {
		t.Fatalf("verbose preflight heading leaked into rebuild:\n%s", got)
	}

	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr:\n%s", stderr.String())
	}
}

func TestRebuildPreflightFailureKeepsDiagnostics(t *testing.T) {
	findings := []installercheck.Finding{
		{
			Level:   "PASS",
			Message: "good check",
		},
		{
			Level:   installercheck.Warn,
			Message: "warning detail",
		},
		{
			Level:   "FAIL",
			Message: "failure detail",
		},
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	status := reportRebuildPreflightResult(
		findings,
		true,
		&stdout,
		&stderr,
	)

	if status != 1 {
		t.Fatalf("status = %d, want 1", status)
	}

	got := stdout.String()

	if strings.Contains(got, "PASS: good check") {
		t.Fatalf("PASS noise leaked into failed rebuild:\n%s", got)
	}

	if !strings.Contains(got, "WARN: warning detail") {
		t.Fatalf("warning detail missing:\n%s", got)
	}

	if !strings.Contains(got, "FAIL: failure detail") {
		t.Fatalf("failure detail missing:\n%s", got)
	}

	if !strings.Contains(
		stderr.String(),
		"FAIL: GjallarOS preflight failed",
	) {
		t.Fatalf(
			"failure summary missing:\n%s",
			stderr.String(),
		)
	}
}
