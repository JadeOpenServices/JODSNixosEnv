package recoveryresize

import (
	"bytes"
	"context"
	"os"
	"os/exec"
)

type systemRunner struct{}

var geteuid = os.Geteuid

// command is exec.CommandContext, except that sudo is dropped when the
// process already runs as root: the recovery-storage maintenance initrd
// has no sudo on PATH (e2e-target, 2026-10-05: `exec: "sudo": executable
// file not found in $PATH`).
func command(
	ctx context.Context,
	name string,
	args ...string,
) *exec.Cmd {
	if name == "sudo" && geteuid() == 0 {
		for len(args) > 0 && (args[0] == "-n" || args[0] == "--") {
			args = args[1:]
		}
		if len(args) > 0 {
			name, args = args[0], args[1:]
		}
	}
	return exec.CommandContext(ctx, name, args...)
}

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
	return command(ctx, name, args...).Output()
}

func (systemRunner) Run(
	ctx context.Context,
	name string,
	args ...string,
) error {
	return command(ctx, name, args...).Run()
}

func (systemRunner) RunInput(
	ctx context.Context,
	input []byte,
	name string,
	args ...string,
) error {
	cmd := command(ctx, name, args...)
	cmd.Stdin = bytes.NewReader(input)
	return cmd.Run()
}

func SystemRunner() ExecutorRunner {
	return systemRunner{}
}
