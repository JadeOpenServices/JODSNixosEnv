package oddccli

import (
	"flag"
	"fmt"
	"io"
	"os"
)

// Nix runs `nix flake update`; Gjallarctl, when set, runs the rebuild
// instead of this gjallarctl.
var (
	Nix        = "nix"
	Gjallarctl = ""
)

// runUpdate moves the checkout's oddc input to ODDC's current main. The
// next rebuild fetches this device's answer at that commit; --rebuild
// runs it now.
func runUpdate(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gjallarctl oddc update", flag.ContinueOnError)
	flags.SetOutput(stderr)

	repo := flags.String("repo", ".", "GjallarOS repository root")
	rebuild := flags.Bool("rebuild", false, "rebuild after updating the oddc input")

	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "Usage: gjallarctl oddc update [--repo PATH] [--rebuild]")
		return 2
	}

	if code := execute(stdout, stderr, "nix", Nix, "flake", "update", "oddc", "--flake", *repo); code != 0 {
		return code
	}
	if !*rebuild {
		fmt.Fprintln(stdout, "[GjallarOS] oddc input updated; the next rebuild fetches this device's ODDC answer at the new commit")
		return 0
	}

	self := Gjallarctl
	if self == "" {
		var err error
		if self, err = os.Executable(); err != nil {
			fmt.Fprintf(stderr, "ERROR: find gjallarctl: %v\n", err)
			return 1
		}
	}

	return execute(stdout, stderr, "gjallarctl", self, "rebuild", "--repo", *repo)
}
