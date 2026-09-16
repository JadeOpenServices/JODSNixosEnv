package oddcvalidation

import (
	"testing"
	"time"
)

func completeReport() Report {
	results := make([]Result, 0, len(RequiredGates))
	for _, gate := range RequiredGates {
		results = append(results, Result{
			Gate:   gate,
			Passed: true,
		})
	}

	return Report{Results: results}
}

func TestValidationMetadataRejectsIncompleteReport(t *testing.T) {
	report := Report{
		Results: []Result{
			{
				Gate:   GateGoTests,
				Passed: true,
			},
		},
	}

	_, err := ValidationMetadata(
		report,
		DeviceContext{},
		"26.05",
		time.Now(),
	)

	if err == nil {
		t.Fatal("incomplete validation report produced metadata")
	}
}
