package oddcvalidation

import (
	"context"
	"testing"
)

func TestRunStopsBeforeDeviceStageWhenStaticGateFails(t *testing.T) {
	deviceStageCalled := false

	staticStage := func(context.Context, string, CommandRunner) Report {
		return Report{Results: []Result{
			{
				Gate:    GateInstallerCheck,
				Passed:  false,
				Details: "project WARN present",
			},
		}}
	}

	deviceStage := func(
		context.Context,
		string,
		CommandRunner,
	) (Report, DeviceContext, error) {
		deviceStageCalled = true
		return Report{}, DeviceContext{}, nil
	}

	_, _, err := run(
		context.Background(),
		"/repo",
		nil,
		staticStage,
		deviceStage,
	)

	if err == nil {
		t.Fatal("failed static validation was accepted")
	}
	if deviceStageCalled {
		t.Fatal("device validation stage ran after static gate failure")
	}
}
