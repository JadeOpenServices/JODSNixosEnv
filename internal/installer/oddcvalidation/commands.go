package oddcvalidation

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type CommandRunner func(
	ctx context.Context,
	dir string,
	name string,
	args ...string,
) ([]byte, error)

func defaultCommandRunner(
	ctx context.Context,
	dir string,
	name string,
	args ...string,
) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}

func runCommandGate(
	ctx context.Context,
	runner CommandRunner,
	repo string,
	gate Gate,
	name string,
	args ...string,
) Result {
	output, err := runner(ctx, repo, name, args...)
	if err != nil {
		details := strings.TrimSpace(string(output))
		if details == "" {
			details = err.Error()
		} else {
			details = fmt.Sprintf("%v: %s", err, details)
		}
		return Result{
			Gate:    gate,
			Passed:  false,
			Details: details,
		}
	}

	return Result{
		Gate:   gate,
		Passed: true,
	}
}

func runFocusedODDCTests(
	ctx context.Context,
	runner CommandRunner,
	repo string,
) Result {
	return runCommandGate(
		ctx,
		runner,
		repo,
		GateFocusedODDCTests,
		"go",
		"test",
		"./internal/installer/oddc",
		"./internal/installer/oddcvalidation",
	)
}

func runAllGoTests(
	ctx context.Context,
	runner CommandRunner,
	repo string,
) Result {
	return runCommandGate(
		ctx,
		runner,
		repo,
		GateGoTests,
		"go",
		"test",
		"./...",
	)
}

func runDeviceEvaluation(
	ctx context.Context,
	runner CommandRunner,
	repo string,
	hostname string,
) Result {
	return runCommandGate(
		ctx,
		runner,
		repo,
		GateDeviceEvaluation,
		"nix",
		"eval",
		"--raw",
		"path:.#nixosConfigurations."+hostname+".config.system.build.toplevel.drvPath",
	)
}

func runRealMachineRebuild(
	ctx context.Context,
	runner CommandRunner,
	repo string,
	executable string,
	hostname string,
) Result {
	return runCommandGate(
		ctx,
		runner,
		repo,
		GateRealMachineRebuild,
		executable,
		"rebuild",
		"--repo",
		repo,
		"--host",
		hostname,
		"--no-cleanup",
	)
}

func runFlakeCheck(
	ctx context.Context,
	runner CommandRunner,
	repo string,
) Result {
	return runCommandGate(
		ctx,
		runner,
		repo,
		GateFlakeCheck,
		"nix",
		"flake",
		"check",
		"--no-build",
		"--all-systems",
		"path:.",
	)
}
