package oddccli

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/bakanura/gjallarOS/internal/installer/deviceprofile"
	"github.com/bakanura/gjallarOS/internal/installer/oddc"
	"github.com/bakanura/gjallarOS/internal/installercheck"
)

// Nix runs `nix flake update`; Gjallarctl, when set, runs the rebuild
// instead of this gjallarctl.
var (
	Nix        = "nix"
	Gjallarctl = ""
)

// oddcState is what the checkout says about ODDC at one moment.
type oddcState struct {
	Locked string // commit flake.lock pins
	Answer string // commit the answer came from; "" without one
}

func readODDCState(repo string) (oddcState, error) {
	locked, err := oddc.LockedRevision(repo)
	if err != nil {
		return oddcState{}, err
	}

	return oddcState{
		Locked: locked,
		Answer: oddc.AnswerRevision(answerDir(repo)),
	}, nil
}

func answerDir(repo string) string {
	return filepath.Join(repo, filepath.FromSlash(deviceprofile.AnswerDir))
}

func short(rev string) string {
	if len(rev) > 12 {
		return rev[:12]
	}
	return rev
}

// lockSummary says whether the oddc input moved.
func lockSummary(before, after oddcState) string {
	if before.Locked == after.Locked {
		return "ODDC: already at " + short(after.Locked)
	}
	return "ODDC: " + short(before.Locked) + " -> " + short(after.Locked)
}

// answerSummary says where this machine's answer stands against the lock.
func answerSummary(state oddcState) string {
	switch state.Answer {
	case "":
		return "ODDC answer: none (no ODDC model matches this machine)"
	case state.Locked:
		return "ODDC answer: at " + short(state.Answer)
	default:
		return "ODDC answer: still at " + short(state.Answer) + ", lock pins " + short(state.Locked)
	}
}

func runUpdate(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gjallarctl oddc update", flag.ContinueOnError)
	flags.SetOutput(stderr)

	repo := flags.String("repo", "", "GjallarOS repository root (default: the system's checkout)")
	rebuild := flags.Bool("rebuild", false, "rebuild after updating the oddc input")
	simple := flags.Bool("simple", false, "print only the result line")
	debug := flags.Bool("debug", false, "print the ODDC state before and after")

	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 || (*simple && *debug) {
		fmt.Fprintln(stderr, "Usage: gjallarctl oddc update [--repo PATH] [--rebuild] [--simple|--debug]")
		return 2
	}

	root, err := installercheck.DiscoverRepository(*repo)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: resolve GjallarOS repository: %v\n", err)
		return 2
	}

	before, err := readODDCState(root)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}

	// --simple keeps tool output for failures only.
	toolOut, toolErr := stdout, stderr
	var captured bytes.Buffer
	if *simple {
		toolOut, toolErr = &captured, &captured
	}

	if code := execute(toolOut, toolErr, "nix", Nix, "flake", "update", "oddc", "--flake", root); code != 0 {
		io.Copy(stderr, &captured)
		return code
	}
	afterLock, err := readODDCState(root)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}

	after, code := afterLock, 0
	if *rebuild {
		self := Gjallarctl
		if self == "" {
			if self, err = os.Executable(); err != nil {
				fmt.Fprintf(stderr, "ERROR: find gjallarctl: %v\n", err)
				return 1
			}
		}
		if *simple {
			// The rebuild asks for authentication on stderr; keep it.
			toolOut, toolErr = io.Discard, stderr
		}
		code = execute(toolOut, toolErr, "gjallarctl", self, "rebuild", "--repo", root)
		if after, err = readODDCState(root); err != nil {
			fmt.Fprintf(stderr, "ERROR: %v\n", err)
			return 1
		}
	}

	switch {
	case *simple:
		line := lockSummary(before, afterLock)
		if *rebuild {
			line += "; " + answerSummary(after)
			if code != 0 {
				line += "; rebuild FAILED"
			}
		}
		fmt.Fprintln(stdout, line)
	case *debug:
		model, err := oddc.AnswerModel(answerDir(root))
		if after.Answer == "" {
			model = "none"
		} else if err != nil {
			model = "unreadable: " + err.Error()
		}
		fmt.Fprintf(stdout, "[GjallarOS] ODDC update\n")
		fmt.Fprintf(stdout, "  repository:    %s\n", root)
		fmt.Fprintf(stdout, "  lock before:   %s\n", before.Locked)
		fmt.Fprintf(stdout, "  lock after:    %s\n", after.Locked)
		fmt.Fprintf(stdout, "  answer before: %s\n", orNone(before.Answer))
		fmt.Fprintf(stdout, "  answer after:  %s\n", orNone(after.Answer))
		fmt.Fprintf(stdout, "  model:         %s\n", model)
		fmt.Fprintf(stdout, "  answer action: %s\n", answerAction(before, after, *rebuild))
		if *rebuild {
			fmt.Fprintf(stdout, "  rebuild exit:  %d\n", code)
		}
	default:
		fmt.Fprintln(stdout, "[GjallarOS] "+lockSummary(before, afterLock))
		if *rebuild {
			fmt.Fprintln(stdout, "[GjallarOS] "+answerSummary(after))
		} else if after.Answer != "" && after.Answer != after.Locked {
			fmt.Fprintln(stdout, "[GjallarOS] The next rebuild fetches this device's ODDC answer at "+short(after.Locked))
		}
	}

	return code
}

func orNone(rev string) string {
	if rev == "" {
		return "none"
	}
	return rev
}

// answerAction names what happened to the answer during this update.
func answerAction(before, after oddcState, rebuilt bool) string {
	switch {
	case after.Answer == "":
		return "none: no ODDC model matches this machine"
	case !rebuilt && after.Answer != after.Locked:
		return "pending: the next rebuild fetches it"
	case before.Answer == after.Answer && after.Answer == after.Locked:
		return "unchanged"
	case after.Answer == after.Locked:
		return "fetched"
	default:
		return "kept: fetch failed, see the rebuild warning"
	}
}
