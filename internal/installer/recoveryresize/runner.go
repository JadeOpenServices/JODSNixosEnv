package recoveryresize

import (
	"bytes"
	"context"
	"os/exec"
)

type systemRunner struct{}

func privilegedReadOnlyCommand(
	name string,
	args []string,
) (string, []string) {
	switch name {
	case "cryptsetup", "blockdev", "btrfs":
		sudoArgs := []string{"-n", "--", name}
		sudoArgs = append(sudoArgs, args...)
		return "sudo", sudoArgs
	default:
		return name, args
	}
}

func (systemRunner) Output(
	ctx context.Context,
	name string,
	args ...string,
) ([]byte, error) {
	name, args = privilegedReadOnlyCommand(name, args)
	return exec.CommandContext(ctx, name, args...).Output()
}

func (systemRunner) Run(
	ctx context.Context,
	name string,
	args ...string,
) error {
	return exec.CommandContext(ctx, name, args...).Run()
}

func (systemRunner) RunInput(
	ctx context.Context,
	input []byte,
	name string,
	args ...string,
) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = bytes.NewReader(input)
	return cmd.Run()
}

func SystemRunner() ExecutorRunner {
	return systemRunner{}
}
