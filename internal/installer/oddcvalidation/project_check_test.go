package oddcvalidation

import (
	"strings"
	"testing"

	"github.com/bakanura/gjallarOS/internal/installercheck"
)

func TestProjectCheckAcceptsZeroWarnings(t *testing.T) {
	result := projectCheckResult(installercheck.Report{
		Findings: []installercheck.Finding{
			{Level: installercheck.OK, Message: "all good"},
		},
	})

	if !result.Passed {
		t.Fatalf("zero-WARN report rejected: %+v", result)
	}
}

func TestProjectCheckRejectsWarning(t *testing.T) {
	result := projectCheckResult(installercheck.Report{
		Findings: []installercheck.Finding{
			{Level: installercheck.Warn, Message: "project warning"},
		},
	})

	if result.Passed {
		t.Fatal("WARN report was accepted")
	}
	if !strings.Contains(result.Details, "WARN: project warning") {
		t.Fatalf("unexpected details: %q", result.Details)
	}
}

func TestProjectCheckRejectsError(t *testing.T) {
	result := projectCheckResult(installercheck.Report{
		Findings: []installercheck.Finding{
			{Level: installercheck.Error, Message: "project error"},
		},
	})

	if result.Passed {
		t.Fatal("ERROR report was accepted")
	}
}
