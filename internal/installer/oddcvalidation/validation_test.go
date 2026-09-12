package oddcvalidation

import "testing"

func passingReport() Report {
	results := make([]Result, 0, len(RequiredGates))
	for _, gate := range RequiredGates {
		results = append(results, Result{
			Gate:   gate,
			Passed: true,
		})
	}
	return Report{Results: results}
}

func TestReportCompleteRequiresEveryGate(t *testing.T) {
	report := passingReport()
	report.Results = report.Results[:len(report.Results)-1]

	if err := report.Complete(); err == nil {
		t.Fatal("incomplete validation report was accepted")
	}
}

func TestReportCompleteRejectsFailedGate(t *testing.T) {
	report := passingReport()
	report.Results[3].Passed = false
	report.Results[3].Details = "test failure"

	if err := report.Complete(); err == nil {
		t.Fatal("failed validation gate was accepted")
	}
}

func TestReportCompleteRejectsDuplicateGate(t *testing.T) {
	report := passingReport()
	report.Results = append(report.Results, report.Results[0])

	if err := report.Complete(); err == nil {
		t.Fatal("duplicate validation gate was accepted")
	}
}

func TestReportCompleteAcceptsFullSuccess(t *testing.T) {
	if err := passingReport().Complete(); err != nil {
		t.Fatal(err)
	}
}
