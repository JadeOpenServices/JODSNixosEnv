package app

import (
	"context"
	"fmt"
	"os"
	"time"
)

type hardwareLifecycleAction int

const (
	hardwareRetain hardwareLifecycleAction = iota
	hardwareGenerate
	hardwareSkip
)

type hardwareGenerateFunc func(
	context.Context,
	string,
	string,
	time.Time,
) (string, error)

type hardwareLifecycleResult struct {
	action  hardwareLifecycleAction
	existed bool
	backup  string
}

func decideHardwareLifecycle(
	existing bool,
	skipHardware bool,
	refreshHardware bool,
	hardwareExists bool,
) hardwareLifecycleAction {
	if skipHardware {
		return hardwareSkip
	}

	if refreshHardware {
		return hardwareGenerate
	}

	if existing && hardwareExists {
		return hardwareRetain
	}

	return hardwareGenerate
}

func hardwareConfigurationExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err == nil {
		if !info.Mode().IsRegular() {
			return false, fmt.Errorf(
				"hardware configuration is not a regular file: %s",
				path,
			)
		}
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return false, fmt.Errorf(
		"inspect hardware configuration %s: %w",
		path,
		err,
	)
}

func reconcileHardwareConfiguration(
	ctx context.Context,
	repo string,
	target string,
	existing bool,
	skipHardware bool,
	refreshHardware bool,
	now time.Time,
	generate hardwareGenerateFunc,
) (hardwareLifecycleResult, error) {
	exists, err := hardwareConfigurationExists(target)
	if err != nil {
		return hardwareLifecycleResult{}, err
	}

	action := decideHardwareLifecycle(
		existing,
		skipHardware,
		refreshHardware,
		exists,
	)

	result := hardwareLifecycleResult{
		action:  action,
		existed: exists,
	}

	switch action {
	case hardwareSkip:
		if !exists {
			return result, fmt.Errorf(
				"--skip-hardware was requested but generated hardware configuration is missing at %s; refusing to continue to flake validation",
				target,
			)
		}
		return result, nil

	case hardwareRetain:
		return result, nil

	case hardwareGenerate:
		backup, err := generate(ctx, repo, target, now)
		if err != nil {
			return result, err
		}
		result.backup = backup
		return result, nil

	default:
		return result, fmt.Errorf(
			"unknown hardware lifecycle action %d",
			action,
		)
	}
}
