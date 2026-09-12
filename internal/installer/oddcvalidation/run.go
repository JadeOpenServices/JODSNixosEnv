package oddcvalidation

import (
	"context"
	"fmt"
)

type StaticStageRunner func(context.Context, string, CommandRunner) Report

type DeviceStageRunner func(
	context.Context,
	string,
	CommandRunner,
) (Report, DeviceContext, error)

func Run(
	ctx context.Context,
	repo string,
	runner CommandRunner,
) (Report, DeviceContext, error) {
	return run(ctx, repo, runner, RunStatic, RunDeviceStage)
}

func run(
	ctx context.Context,
	repo string,
	runner CommandRunner,
	runStaticStage StaticStageRunner,
	runDeviceStage DeviceStageRunner,
) (Report, DeviceContext, error) {
	static := runStaticStage(ctx, repo, runner)

	if err := firstGateFailure(static); err != nil {
		return static, DeviceContext{}, err
	}

	deviceReport, device, err := runDeviceStage(ctx, repo, runner)
	if err != nil {
		return static, DeviceContext{}, fmt.Errorf(
			"ODDC device validation stage: %w",
			err,
		)
	}

	report := Report{
		Results: append(
			append([]Result(nil), static.Results...),
			deviceReport.Results...,
		),
	}

	if err := report.Complete(); err != nil {
		return report, device, err
	}

	return report, device, nil
}

func firstGateFailure(report Report) error {
	for _, result := range report.Results {
		if result.Passed {
			continue
		}

		if result.Details != "" {
			return fmt.Errorf(
				"ODDC validation gate %q failed: %s",
				result.Gate,
				result.Details,
			)
		}

		return fmt.Errorf(
			"ODDC validation gate %q failed",
			result.Gate,
		)
	}

	return nil
}
