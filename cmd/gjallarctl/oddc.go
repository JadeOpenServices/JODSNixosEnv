package main

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
	"slices"
	"strings"
)

// The Nix package sets both through -ldflags; the defaults suit a
// development checkout.
var (
	oddcctlPath = "oddcctl"
	oddcRoot    = "oddc"
)

// oddcctlCommands pass through to oddcctl, which owns them. gjallarctl only
// points them at the catalog shipped with this system, so every model is
// visible, including ones this machine does not use.
var oddcctlCommands = []string{"validate", "list", "resolve", "explain"}

func runODDCctl(args []string, stdout, stderr io.Writer) int {
	argv := slices.Clone(args)
	if !slices.ContainsFunc(argv, func(arg string) bool {
		return arg == "--root" || strings.HasPrefix(arg, "--root=")
	}) {
		argv = append(argv, "--root", oddcRoot)
	}

	cmd := exec.Command(oddcctlPath, argv...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return exit.ExitCode()
		}
		fmt.Fprintf(stderr, "ERROR: run oddcctl: %v\n", err)
		return 1
	}
	return 0
}

func isODDCctlCommand(name string) bool {
	return slices.Contains(oddcctlCommands, name)
}
