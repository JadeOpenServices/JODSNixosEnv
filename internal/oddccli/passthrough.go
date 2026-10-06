package oddccli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// Path is the `oddc` command, which the ODDC NixOS module installs.
var Path = "oddc"

// runODDC runs `oddc ARGS...`. On a deployed system oddc itself defaults
// to /etc/oddc, the deployed model and its host overlay.
func runODDC(args []string, stdout, stderr io.Writer) int {
	cmd := exec.Command(Path, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return exit.ExitCode()
		}
		fmt.Fprintf(stderr, "ERROR: run oddc: %v\n", err)
		return 1
	}

	return 0
}
