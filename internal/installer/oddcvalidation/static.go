package oddcvalidation

import (
	"context"
	"path/filepath"

	"github.com/bakanura/gjallarOS/internal/installer/config"
	"github.com/bakanura/gjallarOS/internal/installercheck"
)

type UserLoader func(path string) (config.User, error)

func RunStatic(
	ctx context.Context,
	repo string,
	runner CommandRunner,
) Report {
	return runStatic(ctx, repo, runner, config.Load)
}

func runStatic(
	ctx context.Context,
	repo string,
	runner CommandRunner,
	loadUser UserLoader,
) Report {
	if runner == nil {
		runner = defaultCommandRunner
	}

	user, err := loadUser(filepath.Join(repo, "user.config.json"))
	if err != nil {
		return Report{Results: []Result{
			{
				Gate:    GateDeviceEvaluation,
				Passed:  false,
				Details: err.Error(),
			},
		}}
	}

	results := []Result{
		runFocusedODDCTests(ctx, runner, repo),
		runAllGoTests(ctx, runner, repo),
		projectCheckResult(installercheck.Check(ctx, repo)),
		runFlakeCheck(ctx, runner, repo),
		runDeviceEvaluation(ctx, runner, repo, user.Hostname),
	}

	return Report{Results: results}
}
