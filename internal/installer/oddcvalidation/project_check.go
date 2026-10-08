package oddcvalidation

import (
	"strings"

	"github.com/JadeOpenServices/gjallarOS/internal/installercheck"
)

func projectCheckResult(report installercheck.Report) Result {
	var problems []string

	for _, finding := range report.Findings {
		switch finding.Level {
		case installercheck.Error, installercheck.Warn:
			problems = append(
				problems,
				string(finding.Level)+": "+finding.Message,
			)
		}
	}

	if len(problems) != 0 {
		return Result{
			Gate:    GateInstallerCheck,
			Passed:  false,
			Details: strings.Join(problems, "; "),
		}
	}

	return Result{
		Gate:   GateInstallerCheck,
		Passed: true,
	}
}
